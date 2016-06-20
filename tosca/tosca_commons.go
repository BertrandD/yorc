package tosca

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"sync"
)

type ValueAssignment struct {
	Expression *TreeNode
}

func (p ValueAssignment) String() string {
	return p.Expression.String()
}

func parseExprNode(value interface{}, t *TreeNode) error {
	switch v := value.(type) {
	case string:
		log.Printf("Found string value %v %T", v, v)
		t.Add(v)
	case []interface{}:
		log.Printf("Found array value %v %T", v, v)
		for _, tabVal := range v {
			log.Printf("Found sub expression node %v %T", tabVal, tabVal)
			if err := parseExprNode(tabVal, t); err != nil {
				return err
			}
		}
	case map[interface{}]interface{}:
		log.Printf("Found map value %v %T", v, v)
		c, err := parseExpression(v)
		if err != nil {
			return err
		}
		t.AddChild(c)
	default:
		return fmt.Errorf("Unexpected type for expression element %T", v)
	}
	return nil
}

func parseExpression(e map[interface{}]interface{}) (*TreeNode, error) {
	if len(e) != 1 {
		return nil, fmt.Errorf("Expecting only one element in expression found %d", len(e))
	}
	log.Printf("parsing %+v", e)
	for key, value := range e {
		keyS, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("Expecting a string for key element '%+v'", key)
		}
		log.Printf("Found expression node with name '%s' and value '%+v' (type '%T')", keyS, value, value)
		t := newTreeNode(keyS)
		err := parseExprNode(value, t)
		return t, err
	}
	return nil, fmt.Errorf("Missing element in expression %s", e)
}

func (p *ValueAssignment) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		p.Expression = newTreeNode(s)
		return nil
	}
	var m map[interface{}]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}
	expr, err := parseExpression(m)
	if err != nil {
		return err
	}
	p.Expression = expr
	return nil
}

type TreeNode struct {
	Value    string
	parent   *TreeNode
	children []*TreeNode
	lock     sync.Mutex
}

func newTreeNode(value string) *TreeNode {
	return &TreeNode{Value: value, children: make([]*TreeNode, 0)}
}
func (t *TreeNode) AddChild(child *TreeNode) error {
	if child.parent != nil {
		return fmt.Errorf("node %s already have a parent, can't adopt it", child)
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	child.parent = t
	t.children = append(t.children, child)
	return nil
}

func (t *TreeNode) Add(value string) error {
	return t.AddChild(newTreeNode(value))
}

func (t *TreeNode) Parent() *TreeNode {
	return t.parent
}

func (t *TreeNode) SetParent(parent *TreeNode) error {
	if t.parent != nil {
		return fmt.Errorf("node %s already have a parent", t)
	}
	t.parent = parent
	return nil
}

func (t *TreeNode) Children() []*TreeNode {
	return t.children
}

func (t *TreeNode) IsLiteral() bool {
	return len(t.children) == 0
}

func (t *TreeNode) String() string {
	buf := &bytes.Buffer{}
	shouldQuote := strings.ContainsAny(t.Value, ":[],")
	if shouldQuote {
		buf.WriteString("\"")
	}
	buf.WriteString(t.Value)
	if shouldQuote {
		buf.WriteString("\"")
	}
	if t.IsLiteral() {
		return buf.String()
	}
	buf.WriteString(": [")
	for i, c := range t.children {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(c.String())
	}
	buf.WriteString("]")
	return buf.String()
}
