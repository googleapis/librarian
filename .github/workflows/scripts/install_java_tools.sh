#!/bin/bash
# .github/workflows/scripts/install_java_tools.sh
# Downloads and installs Java tools (protoc, gRPC, google-java-format, GAPIC).
#
# How it works:
# JAR tools are installed by downloading the .jar and creating a bash wrapper
# in bin/ that invokes java, allowing them to be used as standard commands.

set -e

# Versions
GRPC_VERSION="1.76.2"
GAPIC_GENERATOR_VERSION="2.66.1"
FORMATTER_VERSION="1.25.2"

# OS and Architecture (Hardcoded for Ubuntu)
OS_ARCHITECTURE="linux-x86_64"

# Directories
# Use a directory that can be added to GITHUB_PATH if running in CI
INSTALL_DIR="${1:-$HOME/java_tools}"
BIN_DIR="$INSTALL_DIR/bin"
mkdir -p "$BIN_DIR"

# Download function
download_from() {
  local url=$1
  local save_as=$2
  echo "Downloading $url..."
  curl -fsSL -o "$save_as" --fail -m 60 --retry 3 "$url"
}

echo "Starting installation of Java tools to $INSTALL_DIR..."

# 2. gRPC Plugin
echo "Installing gRPC plugin..."
GRPC_PLUGIN="$INSTALL_DIR/protoc-gen-java_grpc_bin"
download_from "https://maven-central.storage-download.googleapis.com/maven2/io/grpc/protoc-gen-grpc-java/${GRPC_VERSION}/protoc-gen-grpc-java-${GRPC_VERSION}-${OS_ARCHITECTURE}.exe" "$GRPC_PLUGIN"
chmod +x "$GRPC_PLUGIN"
ln -sf "$GRPC_PLUGIN" "$BIN_DIR/protoc-gen-java_grpc"

# 3. Java Formatter
echo "Installing Java Formatter..."
FORMATTER_JAR="$INSTALL_DIR/google-java-format.jar"
download_from "https://maven-central.storage-download.googleapis.com/maven2/com/google/googlejavaformat/google-java-format/${FORMATTER_VERSION}/google-java-format-${FORMATTER_VERSION}-all-deps.jar" "$FORMATTER_JAR"
cat <<EOF > "$BIN_DIR/google-java-format"
#!/bin/bash
exec java -jar "$FORMATTER_JAR" "\$@"
EOF
chmod +x "$BIN_DIR/google-java-format"

# 4. GAPIC Generator Java
echo "Installing GAPIC Generator Java..."
GAPIC_JAR="$INSTALL_DIR/gapic-generator-java.jar"
download_from "https://maven-central.storage-download.googleapis.com/maven2/com/google/api/gapic-generator-java/${GAPIC_GENERATOR_VERSION}/gapic-generator-java-${GAPIC_GENERATOR_VERSION}.jar" "$GAPIC_JAR"
cat <<EOF > "$BIN_DIR/protoc-gen-java_gapic"
#!/bin/bash
exec java -cp "$GAPIC_JAR" com.google.api.generator.Main "\$@"
EOF
chmod +x "$BIN_DIR/protoc-gen-java_gapic"

echo "--------------------------------------------------"
echo "All tools installed successfully in $BIN_DIR"
echo "--------------------------------------------------"
