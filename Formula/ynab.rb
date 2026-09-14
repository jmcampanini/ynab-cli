class Ynab < Formula
  desc "Read and write a YNAB plan from the terminal"
  homepage "https://github.com/jmcampanini/ynab-cli"
  license "MIT"
  head "https://github.com/jmcampanini/ynab-cli.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X github.com/jmcampanini/ynab-cli/cmd.Version=#{version}
    ]
    system "go", "build", "-buildvcs=false", *std_go_args(ldflags:)
    generate_completions_from_executable(bin/"ynab", "completion")
  end

  test do
    assert_match "ynab version HEAD-", shell_output("#{bin}/ynab --version")
  end
end
