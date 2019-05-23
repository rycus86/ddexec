package xdgopen

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/debug"
	"os"
	"path/filepath"
)

func Register(containerId string, sc *config.StartupConfiguration) {
	if len(sc.XdgOpenMappings) == 0 {
		return
	}

	f, err := os.OpenFile(filepath.Join(control.GetDirectoryToShare(), "xdg_open."+containerId), os.O_WRONLY|os.O_CREATE, os.FileMode(0x777))
	if err != nil && debug.IsEnabled() {
		fmt.Println("Failed to register xdg-open mappings for", containerId, ":", err)
		return
	}
	defer f.Close()

	for key, value := range sc.XdgOpenMappings {
		f.WriteString(key + "=" + value + "\n")
	}
}

func Clear(containerId string) {
	os.Remove(filepath.Join(control.GetDirectoryToShare(), "xdg_open."+containerId))
}

/*

https://github.com/go-ini/ini

/usr/share/applications/google-chrome.desktop:MimeType=text/html;text/xml;application/xhtml_xml;image/webp;x-scheme-handler/http;x-scheme-handler/https;x-scheme-handler/ftp;
Exec=/usr/bin/google-chrome-stable %U

/usr/share/applications/slack.desktop:MimeType=x-scheme-handler/slack;
Exec=/usr/bin/slack %U

*/
