package files

type PasswdFiles struct {
	Passwd string
	Group  string
	Shadow string

	Temporary bool
}
