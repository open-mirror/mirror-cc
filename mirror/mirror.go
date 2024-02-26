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
	"io"
	"os"
)

type Mirror struct {
	files   map[string]string // path -> sha256
	sources []Source
}

func (m *Mirror) Sync(ctx context.Context) {
	keepingAlive := make(map[string]struct{}, len(m.files))

	for _, s := range m.sources {
		logger := io.MultiWriter(os.Stdout, /* log file */)
		syncCtx := NewContext(ctx, logger, s)
		syncCtx.cachedHashes = m.files
		syncCtx.keepingAlive = keepingAlive

		s.Sync(syncCtx)
		syncCtx.cancel(nil)
	}
}
