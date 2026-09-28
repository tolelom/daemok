package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) enums(n *yaml.Node)        {}
func (mg *merger) consts(n *yaml.Node)       {}
func (mg *merger) errors(n *yaml.Node)       {}
func (mg *merger) types(n *yaml.Node)        {}
func (mg *merger) contracts(n *yaml.Node)    {}
func (mg *merger) handlerKinds(n *yaml.Node) {}
