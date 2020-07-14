package actions

// Actionner
type Actionner interface {
	Action(interface{}) (actionned bool, err error)
}
