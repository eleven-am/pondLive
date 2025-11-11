package html

import internalhtml "github.com/eleven-am/pondlive/go/internal/html"

type (
	Item                                         = internalhtml.Item
	Node                                         = internalhtml.Node
	Element                                      = internalhtml.Element
	TextNode                                     = internalhtml.TextNode
	FragmentNode                                 = internalhtml.FragmentNode
	CommentNode                                  = internalhtml.CommentNode
	ComponentNode                                = internalhtml.ComponentNode
	Prop                                         = internalhtml.Prop
	Event                                        = internalhtml.Event
	Modifiers                                    = internalhtml.Modifiers
	Updates                                      = internalhtml.Updates
	EventHandler                                 = internalhtml.EventHandler
	EventOptions                                 = internalhtml.EventOptions
	EventBinding                                 = internalhtml.EventBinding
	AnimationEvent                               = internalhtml.AnimationEvent
	ClickEvent                                   = internalhtml.ClickEvent
	ClipboardEvent                               = internalhtml.ClipboardEvent
	CompositionEvent                             = internalhtml.CompositionEvent
	DialogEvent                                  = internalhtml.DialogEvent
	DragEvent                                    = internalhtml.DragEvent
	FocusEvent                                   = internalhtml.FocusEvent
	FormEvent                                    = internalhtml.FormEvent
	FullscreenEvent                              = internalhtml.FullscreenEvent
	HashChangeEvent                              = internalhtml.HashChangeEvent
	InputEvent                                   = internalhtml.InputEvent
	KeyboardEvent                                = internalhtml.KeyboardEvent
	LifecycleEvent                               = internalhtml.LifecycleEvent
	LoadEvent                                    = internalhtml.LoadEvent
	MediaEvent                                   = internalhtml.MediaEvent
	MouseEvent                                   = internalhtml.MouseEvent
	PointerEvent                                 = internalhtml.PointerEvent
	PrintEvent                                   = internalhtml.PrintEvent
	ResizeEvent                                  = internalhtml.ResizeEvent
	ScrollEvent                                  = internalhtml.ScrollEvent
	SelectionEvent                               = internalhtml.SelectionEvent
	StorageEvent                                 = internalhtml.StorageEvent
	ToggleEvent                                  = internalhtml.ToggleEvent
	TouchEvent                                   = internalhtml.TouchEvent
	TransitionEvent                              = internalhtml.TransitionEvent
	VisibilityEvent                              = internalhtml.VisibilityEvent
	WheelEvent                                   = internalhtml.WheelEvent
	ElementRef[T internalhtml.ElementDescriptor] = internalhtml.ElementRef[T]
	RefListener                                  = internalhtml.RefListener
	Op                                           = internalhtml.Op
	SetText                                      = internalhtml.SetText
	SetAttrs                                     = internalhtml.SetAttrs
	ListOp                                       = internalhtml.ListOp
	ListChildOp                                  = internalhtml.ListChildOp
	Ins                                          = internalhtml.Ins
	Del                                          = internalhtml.Del
	Mov                                          = internalhtml.Mov
	Set                                          = internalhtml.Set
	MetaTag                                      = internalhtml.MetaTag
	LinkTag                                      = internalhtml.LinkTag
	ScriptTag                                    = internalhtml.ScriptTag
	UploadBinding                                = internalhtml.UploadBinding
)
