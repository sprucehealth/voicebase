package main

import (
	"errors"
	"os"
	"syscall"
	"time"
)

func GetStat(fileInfo os.FileInfo) (*Stat, error) {
	sysSt := fileInfo.Sys()
	if sysSt == nil {
		return nil, errors.New("FileInfo.Sys returned NULL Sys")
	}
	st, ok := sysSt.(*syscall.Stat_t)
	if !ok {
		return nil, errors.New("FileInfo.Sys not syscall.Stat_t")
	}

	return &Stat{
		Size:  st.Size,
		Inode: st.Ino,
		Ctime: time.Unix(st.Ctimespec.Unix()).UTC(),
	}, nil
}
