// plex is a multiplexing library for io.Reader, io.Writer, and io.ReadWriter
// types which allows for requesting a reader or writer without needing to know
// which one is actually being used. This is useful for things like managing
// communication over a full-duplex TCP connection using a net.Conn type which
// implements the io.ReadWriter interface.
//
// Along with multiplexing using the io.Reader and io.Writer interfaces, plex
// also supports streaming using the ReadStream, WriteStream, and
// ReadWriteStream interfaces. These interfaces provide access to a single
// stream of data, which can be read or written to one byte at a time.
//
// Most types in plex are implement io.Closer and it is expected that the user
// properly close the streams upon completion using a `defer` statement or
// appropriate closing mechanism to ensure that the streams are returned to
// the pool of available streams.
package plex
