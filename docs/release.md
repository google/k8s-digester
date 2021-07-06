# Releasing

1.  Commit your changes:

    ```sh
    git commit -m "Stuff"
    ```

2.  Create an annotated Git tag of the format `v*-rc*`, e.g., `v0.0.1-rc0`:

    ```sh
    git tag -a v0.0.1-rc0 -m "v0.0.1-rc0"
    ```

3.  Push your commits and the tag:

    ```sh
    git push --follow-tags
    ```

If the release job fails and the release tag isn't created, you can fix the
problem and create a new tag with the same version number, bumping the release
candidate (`rc`) number, e.g. `v0.0.1-rc1`.
