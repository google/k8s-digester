// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/google/k8s-digester/pkg/logging"
)

var (
	ctx     = context.Background()
	nullLog = logging.CreateDiscardLogger()
	log     = logging.CreateStdLogger("handler_test").V(2)
)

func Test_Handle_NoPatchesForDelete(t *testing.T) {
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: "default",
			Object: runtime.RawExtension{
				Raw: []byte(`{}`),
			},
			Operation: admissionv1.Delete,
		},
	}
	h := &Handler{Log: log}

	resp := h.Handle(ctx, req)

	assertAdmissionAllowed(t, resp)
	assertReason(t, resp, reasonNoMutationForOperation)
	assertNoPatches(t, resp)
}

func Test_Handle_DisallowedOnParseError(t *testing.T) {
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte("\U0001f4a9"),
			},
			Operation: admissionv1.Create,
		},
	}
	h := &Handler{Log: nullLog} // suppress output of expected error

	resp := h.Handle(ctx, req)

	assertAdmissionError(t, resp)
}

func Test_Handle_IgnoreError(t *testing.T) {
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: []byte("\U0001f4a9"),
			},
			Operation: admissionv1.Create,
		},
	}
	h := &Handler{
		Log:          nullLog, // suppress output of expected error
		IgnoreErrors: true,
	}

	resp := h.Handle(ctx, req)

	assertAdmissionAllowed(t, resp)
	assertReason(t, resp, reasonErrorIgnored)
	assertNoPatches(t, resp)
}

func Test_Handle_NoMutationOfDigesterNamespaceRequests(t *testing.T) {
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: "digester-system",
			Object: runtime.RawExtension{
				Raw: []byte(`{}`),
			},
			Operation: admissionv1.Create,
		},
	}
	h := &Handler{Log: log}

	resp := h.Handle(ctx, req)

	assertAdmissionAllowed(t, resp)
	assertReason(t, resp, reasonNoSelfManagement)
	assertNoPatches(t, resp)
}

func Test_Handle_NotPatchedWhenNoChange(t *testing.T) {
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: "test",
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: []byte(`{}`),
			},
		},
	}
	resolveImageTags = func(_ context.Context, _ logr.Logger, _ *rest.Config, _ *yaml.RNode, _ []string) error {
		return nil
	}
	h := &Handler{Log: log}

	resp := h.Handle(ctx, req)

	assertAdmissionAllowed(t, resp)
	assertReason(t, resp, reasonNotPatched)
	assertNoPatches(t, resp)
}

func Test_Handle_Patch(t *testing.T) {
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Namespace: "test",
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: []byte(`{"spec": {"containers": [{"image": "registry.example.com/repository/image:tag"}]}}`),
			},
		},
	}
	imageWithDigest := "registry.example.com/repository/image:tag@sha256:digest"
	resolveImageTags = func(_ context.Context, _ logr.Logger, _ *rest.Config, n *yaml.RNode, _ []string) error {
		return n.PipeE(yaml.Lookup("spec", "containers", "0", "image"), yaml.FieldSetter{StringValue: imageWithDigest})
	}
	h := &Handler{Log: log}

	resp := h.Handle(ctx, req)

	assertAdmissionAllowed(t, resp)
	assertReason(t, resp, reasonPatched)
	if len(resp.Patches) < 1 {
		t.Errorf("expected len(resp.Patches) == 1, got %d", len(resp.Patches))
	}
	if diff := cmp.Diff(resp.Patches, []jsonpatch.Operation{
		jsonpatch.NewOperation("replace", "/spec/containers/0/image", imageWithDigest),
	}); diff != "" {
		t.Errorf("patch mismatch (-want +got):\n%s", diff)
	}
}

func assertAdmissionAllowed(t *testing.T, resp admission.Response) {
	if !resp.Allowed {
		t.Errorf("wanted allowed, got disallowed")
	}
	if resp.Result.Code != http.StatusOK {
		t.Logf("result message: %s", resp.Result.Message)
		t.Errorf("wanted code %d, got %d", http.StatusOK, resp.Result.Code)
	}
}

func assertAdmissionError(t *testing.T, resp admission.Response) {
	if resp.Allowed {
		t.Errorf("wanted disallowed, got allowed")
	}
	if resp.Result.Code != http.StatusInternalServerError {
		t.Errorf("wanted code %d, got %d", http.StatusInternalServerError, resp.Result.Code)
	}
	if resp.Result.Message == "" {
		t.Errorf("wanted result message, got empty string")
	}
}

func assertNoPatches(t *testing.T, resp admission.Response) {
	if len(resp.Patch) > 0 {
		t.Errorf("wanted empty Patch field, got %s", resp.Patch)
	}
	if len(resp.Patches) > 0 {
		t.Errorf("wanted empty Patches field, got %+v", resp.Patches)
	}
}

func assertReason(t *testing.T, resp admission.Response, wantReason string) {
	if resp.Result.Reason != metav1.StatusReason(wantReason) {
		t.Errorf("wanted reason %s, got %s", wantReason, resp.Result.Reason)
	}
}
