package unforker

type Unforker struct {
}

func NewUnforker() *Unforker {
	return &Unforker{}
}

func (u *Unforker) FindAndListForksSync() error {
	return nil
}
