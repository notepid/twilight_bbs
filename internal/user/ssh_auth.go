package user

// SSHAuthenticator adapts Repo to be used as an SSH password authenticator.
type SSHAuthenticator struct {
	repo *Repo
}

// NewSSHAuthenticator creates an SSH authenticator from a user repository.
func NewSSHAuthenticator(repo *Repo) *SSHAuthenticator {
	return &SSHAuthenticator{repo: repo}
}

// Authenticate validates username/password for SSH authentication.
func (a *SSHAuthenticator) Authenticate(username, password string) (bool, error) {
	return a.repo.AuthenticateForSSH(username, password)
}
