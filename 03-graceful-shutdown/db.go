package server

import (
	"errors"
	"log/slog"
	"net"
)

type DBPool struct {
	conns  []net.Conn
	logger *slog.Logger
}

func NewDBPool(size int, logger *slog.Logger) *DBPool {
	conns := make([]net.Conn, 0, size)
	for range size {
		p1, _ := net.Pipe()
		conns = append(conns, p1)
	}
	return &DBPool{conns: conns, logger: logger}
}

func (db *DBPool) Close() error {
	errs := make([]error, 0, len(db.conns))
	for _, conn := range db.conns {
		errs = append(errs, conn.Close())
	}

	db.logger.Info("db pool closed", "conns", len(db.conns))

	return errors.Join(errs...)
}
