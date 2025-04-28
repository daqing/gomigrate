gomigrate
=========

1. Running migrations:

    ```bash
    gomigrate migrate [/path/to/migrations/folder] [VERSION]
    ```

    if VERSION is not given, it will run all migrations.

2. Rollback migrations:

    ```bash
    gomigrate rollback [/path/to/migrations/folder] [STEP]
    ```

    if STEP is not given, it will be 1 by default.

