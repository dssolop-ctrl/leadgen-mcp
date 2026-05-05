package direct

import "testing"

func TestResolveImageHostPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"already container abs", "/app/previews/omsk/v1.png", "/app/previews/omsk/v1.png"},
		{"app root", "/app/foo/bar.png", "/app/foo/bar.png"},
		{"relative host (forward)", "docs/campaign_previews/omsk_rsya/v1.png", "/app/previews/omsk_rsya/v1.png"},
		{"relative host with ./", "./docs/campaign_previews/omsk_rsya/v1.png", "/app/previews/omsk_rsya/v1.png"},
		{"windows backslashes", "docs\\campaign_previews\\omsk_rsya\\v1.png", "/app/previews/omsk_rsya/v1.png"},
		{"windows abs", "C:/git/leadgen-mcp/docs/campaign_previews/omsk/v1.png", "/app/previews/omsk/v1.png"},
		{"windows abs backslash", "C:\\git\\leadgen-mcp\\docs\\campaign_previews\\omsk\\v1.png", "/app/previews/omsk/v1.png"},
		{"linux abs host", "/home/user/leadgen-mcp/docs/campaign_previews/omsk/v1.png", "/app/previews/omsk/v1.png"},
		{"unknown path passthrough", "/tmp/foo.png", "/tmp/foo.png"},
		{"random relative passthrough", "some/other/path.png", "some/other/path.png"},
		{"deep nested", "docs/campaign_previews/very/deep/nested/path/img.png", "/app/previews/very/deep/nested/path/img.png"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveImageHostPath(c.in)
			if got != c.want {
				t.Errorf("resolveImageHostPath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
