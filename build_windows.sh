# Build your Go application for Windows
GOOS=windows GOARCH=amd64 go build -o app.exe

# Create the directory structure for the installer
mkdir installer
cp app.exe installer/
cp -r lib/ installer/  # Copy libraries

# Generate the MSI installer
candle ./wix/product.wxs -out installer/app.wixobj
light installer/app.wixobj -o installer/YourAppInstaller.msi

# Clean up temporary files
rm -rf installer/app.wixobj

echo "MSI installer created: YourAppInstaller.msi"
