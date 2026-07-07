package activities

// Sandbox is the full capability surface the tool adapter layer depends on —
// every typed Activity's dependency, combined. Grows as new Activities are
// added; a concrete sandbox implementation satisfies it structurally.
type Sandbox interface {
	FileReader
	FileWriter
	CommandExecutor
}
