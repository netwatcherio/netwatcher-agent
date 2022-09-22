echo Building go...
go build -o ./bin/windows
echo Building .msi
go-msi make --msi NetWatcher-Agent.msi --path ./wix --src ./wix --out ./bin/windows --version 1.0.1