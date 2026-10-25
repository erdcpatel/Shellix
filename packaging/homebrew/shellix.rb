class Shellix < Formula
  desc "Blazing-fast developer terminal workbench built in pure Go"
  homepage "https://github.com/erdcpatel/Shellix"
  version "0.1.0"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/erdcpatel/Shellix/releases/download/v#{version}/shellix_#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_DARWIN_ARM64"
    else
      url "https://github.com/erdcpatel/Shellix/releases/download/v#{version}/shellix_#{version}_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/erdcpatel/Shellix/releases/download/v#{version}/shellix_#{version}_linux_arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_LINUX_ARM64"
    else
      url "https://github.com/erdcpatel/Shellix/releases/download/v#{version}/shellix_#{version}_linux_amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256_LINUX_AMD64"
    end
  end

  def install
    bin.install "shellix"
    bin.install_symlink "shellix" => "sx"
  end

  test do
    assert_match "Shellix", shell_output("#{bin}/sx version")
  end
end
