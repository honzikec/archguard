package astfacts

type ClassDecl struct {
	Name string
	Line int
}

type ImportBinding struct {
	Module    string
	Default   string
	Namespace string
	Named     map[string]string // local -> imported
	Line      int
}

type NewExpression struct {
	ClassName       string
	Line            int
	Column          int
	Raw             string
	IsIdentifier    bool
	ConstructorKind string
}

type FileFacts struct {
	FilePath             string
	Classes              []ClassDecl
	Imports              []ImportBinding
	NewExprs             []NewExpression
	ExportedClassByName  map[string]string // export name -> local class name
	DefaultExportedClass string
}
