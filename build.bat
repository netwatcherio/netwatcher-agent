echo Building go...
go build -o ./bin/windows
echo Building .msi
go-msi make --msi ./bin/windows/netwatcher-agent_install.msi --src ./wix --version 1.0.2