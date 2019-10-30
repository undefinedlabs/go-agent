package sql

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"reflect"
	"strings"
)

type (
	// instrumented driver wrapper
	instrumentedDriver struct {
		driver        driver.Driver
		configuration *driverConfiguration
	}

	driverConfiguration struct {
		t             opentracing.Tracer
		statements    bool
		connString    string
		componentName string
		peerService   string
		user          string
		port          string
		instance      string
		host          string
	}

	Option func(*instrumentedDriver)
)

// Enable statement instrumentation
func WithStatements() Option {
	return func(d *instrumentedDriver) {
		d.configuration.statements = true
	}
}

// Wraps the current sql driver to add instrumentation
func WrapDriver(d driver.Driver, options ...Option) driver.Driver {
	wrapper := &instrumentedDriver{
		driver: d,
		configuration: &driverConfiguration{
			t:          instrumentation.Tracer(),
			statements: false,
		},
	}
	for _, option := range options {
		option(wrapper)
	}
	return wrapper
}

// Open returns a new connection to the database.
// The name is a string in a driver-specific format.
//
// Open may return a cached connection (one previously
// closed), but doing so is unnecessary; the sql package
// maintains a pool of idle connections for efficient re-use.
//
// The returned connection is only used by one goroutine at a
// time.
func (w *instrumentedDriver) Open(name string) (driver.Conn, error) {
	conn, err := w.driver.Open(name)
	if err != nil {
		return nil, err
	}
	w.fillDriverData(name)
	return &instrumentedConn{conn: conn, configuration: w.configuration}, nil
}

// namedValueToValue converts driver arguments of NamedValue format to Value format. Implemented in the same way as in
// database/sql ctxutil.go.
func namedValueToValue(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))
	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}
		dargs[n] = param.Value
	}
	return dargs, nil
}

// newSpan creates a new opentracing.Span instance from the given context.
func (t *driverConfiguration) newSpan(operationName string, query string, args []driver.NamedValue, c *driverConfiguration, ctx context.Context) opentracing.Span {
	var opts []opentracing.StartSpanOption
	parent := opentracing.SpanFromContext(ctx)
	if parent != nil {
		opts = append(opts, opentracing.ChildOf(parent.Context()))
	}
	opts = append(opts, opentracing.Tags{
		"db.type":       "sql",
		"component":     c.componentName,
		"span.kind":     "client",
		"db.conn":       c.connString,
		"peer.service":  c.peerService,
		"db.user":       c.user,
		"peer.port":     c.port,
		"db.instance":   c.instance,
		"peer.hostname": c.host,
	})
	if query != "" {
		stIndex := strings.IndexRune(query, ' ')
		var method string
		if stIndex >= 0 {
			method = strings.ToUpper(query[:stIndex])
		}
		opts = append(opts, opentracing.Tags{
			"db.prepare_statement": query,
			"db.method":            method,
		})
		operationName = fmt.Sprintf("%s:%s", c.peerService, method)
	} else {
		operationName = fmt.Sprintf("%s:%s", c.peerService, strings.ToUpper(operationName))
	}
	if args != nil && len(args) > 0 {
		dbParams := map[string]interface{}{}
		for _, item := range args {
			dbParams[item.Name] = map[string]interface{}{
				"type":  reflect.TypeOf(item.Value).String(),
				"value": item.Value,
			}
		}
		opts = append(opts, opentracing.Tags{
			"db.params": dbParams,
		})
	}
	span := t.t.StartSpan(operationName, opts...)
	return span
}

func (w *instrumentedDriver) fillDriverData(name string) {
	w.configuration.connString = name
	w.configuration.componentName = reflect.TypeOf(w.driver).Elem().String()

	// Postgres detection
	if w.configuration.componentName == "pq.Driver" || w.configuration.componentName == "stdlib.Driver" ||
		w.configuration.componentName == "pgsqldriver.postgresDriver" {
		fillPostgresDriverData(name, w)
	}
}
