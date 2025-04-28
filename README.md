gomigrate
=========

1. Running migrations:

    ```bash
    gomigrate migrate [VERSION] [/path/to/migrations/folder]
    ```

    if VERSION is not given, it will run all migrations.

2. Rollback migrations:

    ```bash
    gomigrate rollback [STEP] [/path/to/migrations/folder]
    ```

    if STEP is not given, it will be 1 by default.

3. Check migration status:

    ```bash
    gomigrate status [/path/to/migrations/folder]
    ```

