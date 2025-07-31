package _runtime

type GoFunc struct {
	Id       int
	Pkg      string
	Receiver Var
	Name     string
	Params   []*Var
	Results  []*Var
}

type Var struct {
	Name     string
	Type     string
	BaseType string
}

func TraceFunc(goFuncId int) {

}
