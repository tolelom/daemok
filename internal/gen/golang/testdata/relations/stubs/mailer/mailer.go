package mailer

type Mailer struct{}

func (*Mailer) Send(to string, body string) error { return nil }
