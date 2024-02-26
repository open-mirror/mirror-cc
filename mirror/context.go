/**
 * BmclAPI (Golang Edition)
 * Copyright (C) 2024 Kevin Z <zyxkad@gmail.com>
 * All rights reserved
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as published
 *  by the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.
 *
 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package mirror

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Context struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	logger        *log.Logger
	dbugLog       *log.Logger
	erroLog       *log.Logger
	logDebugFlags map[string]bool

	mux sync.RWMutex
	storagePath string
	cachedHashes map[string]string
	keepingAlive map[string]struct{}
	httpClient  *http.Client
}

func NewContext(ctx context.Context, logger io.Writer, s Source) (c *Context) {
	c = new(Context)
	c.ctx, c.cancel = context.WithCancelCause(ctx)
	c.newLogger(logger)
	c.setSource(s)
	return
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) newLogger(w io.Writer) {
	c.logger = log.New(w, "[INFO] ", log.Ldate|log.Ltime)
	c.dbugLog = log.New(w, "[DBUG] ", log.Ldate|log.Ltime)
	c.erroLog = log.New(w, "[ERRO] ", log.Ldate|log.Ltime)
}

func (c *Context) setSource(s Source) {
	id := s.Id()
	c.logger.SetPrefix("[INFO/" + id + "] ")
	c.dbugLog.SetPrefix("[DBUG/" + id + "] ")
	c.erroLog.SetPrefix("[ERRO/" + id + "] ")
	c.logDebugFlags = s.Debug()
}

func (c *Context) Debugging(flag string) bool {
	return c.logDebugFlags[flag]
}

func (c *Context) Log(args ...any) {
	c.logger.Println(args...)
}

func (c *Context) Logf(format string, args ...any) {
	c.logger.Printf(format, args...)
}

func (c *Context) Debug(flag string, args ...any) {
	if c.Debugging(flag) {
		c.dbugLog.Println(args...)
	}
}

func (c *Context) Debugf(flag string, format string, args ...any) {
	if c.Debugging(flag) {
		c.dbugLog.Printf(format, args...)
	}
}

func (c *Context) Error(args ...any) {
	c.erroLog.Println(args...)
}

func (c *Context) Errorf(format string, args ...any) {
	c.erroLog.Printf(format, args...)
}

func (c *Context) AbortWithErr(err error) {
	c.Errorf("Aborted: %v", err)
	c.cancel(err)
}

func (c *Context) Aborted() bool {
	return c.ctx.Err() != nil
}

// Hash will returns the SHA256 hash of the file
func (c *Context) Hash(path string) (string, error) {
	c.mux.RLock()
	h, ok := c.cachedHashes[path]
	c.mux.RUnlock()
	if ok {
		return h, nil
	}

	fd, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	hs := sha256.New()
	if _, err := io.Copy(hs, fd); err != nil {
		return "", err
	}

	var buf [32]byte
	h = hex.EncodeToString(hs.Sum(buf[:0]))

	c.mux.Lock()
	defer c.mux.Unlock()
	c.cachedHashes[path] = h
	return h, nil
}

func (c *Context) Create(path string) (io.WriteCloser, error) {
	return os.Create(filepath.Join(c.storagePath, filepath.FromSlash(path)))
}

// KeepAlive mark the file as not outdated
func (c *Context) KeepAlive(path string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.keepingAlive[path] = struct{}{}
}

func (c *Context) DoHTTP(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}
