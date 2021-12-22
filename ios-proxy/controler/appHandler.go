package controler

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/DeanThompson/syncmap"
	"ctp-ios-proxy/utils"
	"path/filepath"
)

func TempFileName(dir, suffix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(dir, hex.EncodeToString(randBytes)+suffix)
}

var background = &utils.Background{
	Sm: syncmap.New(),
}

func forceInstallAPK(filepath string, uid string) error {
	am := &utils.APKManager{Path: filepath}
	am.Uid = uid
	return am.ForceInstall()
}
