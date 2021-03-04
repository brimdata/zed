To create a new migration, install the golang-migrate cli tool:

```
brew install golang-migrate
```

then run:

```
migrate create -ext sql -D <migrations-directory> <name>
```
