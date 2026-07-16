package server

import extension "extension-scaffold/internal/extension"

// StartExtension creates and starts the extension server in a goroutine.
// Returns the extension instance (so callers can wire the FTSO watcher to
// its order store) and an error channel that receives any ListenAndServe
// failure (e.g., port already in use).
func StartExtension(extensionPort, signPort int) (*extension.Extension, <-chan error) {
	e := extension.New(extensionPort, signPort)
	errCh := make(chan error, 1)
	go func() {
		if err := e.Server.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()
	return e, errCh
}
