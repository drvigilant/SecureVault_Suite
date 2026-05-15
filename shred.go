package main

import (
    "crypto/rand"
    "os"
)

func secureShred(path string, passes int) error {
    f, err := os.OpenFile(path, os.O_RDWR, 0)
    if err != nil {
        return err
    }

    info, err := f.Stat()
    if err != nil {
        f.Close()
        return err
    }
    size := info.Size()

    buf := make([]byte, size)
    for i := 0; i < passes; i++ {
        rand.Read(buf)
        f.Seek(0, 0)
        f.Write(buf)
        f.Sync()
    }

    // Final zero pass
    for i := range buf {
        buf[i] = 0
    }
    f.Seek(0, 0)
    f.Write(buf)
    f.Sync()
    f.Close()

    return os.Remove(path)
}
