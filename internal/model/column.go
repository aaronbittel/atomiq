package model

// Column groups work items under one name.
type Column struct {
	name      string
	workItems []WorkItem
}

// NewColumn creates a column with detached work item copies.
func NewColumn(name string, items ...WorkItem) Column {
	clonedItems := make([]WorkItem, len(items))
	for i, item := range items {
		clonedItems[i] = item.clone()
	}
	return Column{
		name:      name,
		workItems: clonedItems,
	}
}

// clone returns a deep copy of c.
func (c Column) clone() Column {
	items := make([]WorkItem, len(c.workItems))
	for i, item := range c.workItems {
		items[i] = item.clone()
	}
	return Column{
		name:      c.name,
		workItems: items,
	}
}
