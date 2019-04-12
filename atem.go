package atem

import (
	"path/filepath"

	"github.com/go-leap/fs"
	"github.com/go-leap/sys"
)

func New(dirPath string) (ctx *Ctx, err error) {
	if dirPath != "" {
		if dirPath[0] == '~' && dirPath[1] == filepath.Separator {
			dirPath = filepath.Join(usys.UserHomeDirPath(), dirPath[2:])
		}
		dirPath, err = filepath.Abs(dirPath)
	}
	if ctx = (&Ctx{Dir: dirPath}); err == nil && dirPath != "" && ufs.IsDir(dirPath) {

	}
	return
}