echo Building go...
go build -o ./bin/windows
echo Building .msi
go-msi make --msi ./bin/windows/netwatcher-agent_install.msi --src ./wix --out ./bin/windows --version 1.0.1