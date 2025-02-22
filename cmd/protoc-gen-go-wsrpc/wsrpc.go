package main

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage = protogen.GoImportPath("context")
	wsrpcPackage   = protogen.GoImportPath("github.com/goplugin/wsrpc")
)

// generateFile generates a _wsrpc.pb.go file containing wsRPC service definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	if len(file.Services) == 0 {
		return nil
	}
	filename := file.GeneratedFilenamePrefix + "_wsrpc.pb.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// Code generated by protoc-gen-go-wsrpc. DO NOT EDIT.")
	g.P("// versions:")
	g.P("// - protoc-gen-go-wsrpc v", version)
	g.P("// - protoc             ", protocVersion(gen))
	g.P()
	g.P("package ", file.GoPackageName)
	g.P()
	generateFileContent(gen, file, g)

	return g
}

func protocVersion(gen *protogen.Plugin) string {
	v := gen.Request.GetCompilerVersion()
	if v == nil {
		return "(unknown)"
	}
	var suffix string
	if s := v.GetSuffix(); s != "" {
		suffix = "-" + s
	}

	return fmt.Sprintf("v%d.%d.%d%s", v.GetMajor(), v.GetMinor(), v.GetPatch(), suffix)
}

// generateFileContent generates the wsrpc service definitions, excluding the package statement.
func generateFileContent(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile) {
	if len(file.Services) == 0 {
		return
	}

	for _, service := range file.Services {
		genService(gen, file, g, service)
	}
}

func genService(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	clientName := service.GoName + "Client"
	g.P("// ", clientName, " is the client API for ", service.GoName, " service.")
	g.P("//")

	// Client interface.
	g.AnnotateSymbol(clientName, protogen.Annotation{Location: service.Location})
	g.P("type ", clientName, " interface {")
	for _, method := range service.Methods {
		g.AnnotateSymbol(clientName+"."+method.GoName, protogen.Annotation{Location: method.Location})
		g.P(method.Comments.Leading,
			clientSignature(g, method))
	}
	g.P("}")
	g.P()

	// Client structure.
	g.P("type ", unexport(clientName), " struct {")
	g.P("cc ", wsrpcPackage.Ident("ClientInterface"))
	g.P("}")
	g.P()

	// NewClient factory.
	g.P("func New", clientName, " (cc ", wsrpcPackage.Ident("ClientInterface"), ") ", clientName, " {")
	g.P("return &", unexport(clientName), "{cc}")
	g.P("}")
	g.P()

	var methodIndex int
	// Client method implementations.
	for _, method := range service.Methods {
		// Unary RPC method
		genClientMethod(gen, file, g, method, methodIndex)
		methodIndex++
	}

	// Server interface.
	serverType := service.GoName + "Server"
	g.P("// ", serverType, " is the server API for ", service.GoName, " service.")
	g.AnnotateSymbol(serverType, protogen.Annotation{Location: service.Location})
	g.P("type ", serverType, " interface {")
	for _, method := range service.Methods {
		g.AnnotateSymbol(serverType+"."+method.GoName, protogen.Annotation{Location: method.Location})
		g.P(method.Comments.Leading,
			serverSignature(g, method))
	}
	g.P("}")
	g.P()

	// Server registration.
	serviceDescVar := service.GoName + "_ServiceDesc"
	g.P("func Register", service.GoName, "Server(s ", wsrpcPackage.Ident("ServiceRegistrar"), ", srv ", serverType, ") {")
	g.P("s.RegisterService(&", serviceDescVar, `, srv)`)
	g.P("}")
	g.P()

	// Server handler implementations.
	var handlerNames []string
	for _, method := range service.Methods {
		hname := genServerMethod(gen, file, g, method)
		handlerNames = append(handlerNames, hname)
	}

	// Service descriptor.
	g.P("// ", serviceDescVar, " is the ", wsrpcPackage.Ident("ServiceDesc"), " for ", service.GoName, " service.")
	g.P("// It's only intended for direct use with ", wsrpcPackage.Ident("RegisterService"), ",")
	g.P("// and not to be introspected or modified (even as a copy)")
	g.P("var ", serviceDescVar, " = ", wsrpcPackage.Ident("ServiceDesc"), " {")
	g.P("ServiceName: ", strconv.Quote(string(service.Desc.FullName())), ",")
	g.P("HandlerType: (*", serverType, ")(nil),")
	g.P("Methods: []", wsrpcPackage.Ident("MethodDesc"), "{")
	for i, method := range service.Methods {
		g.P("{")
		g.P("MethodName: ", strconv.Quote(string(method.Desc.Name())), ",")
		g.P("Handler: ", handlerNames[i], ",")
		g.P("},")
	}
	g.P("},")
	g.P("}")
	g.P()
}

func clientSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	s := method.GoName + "(ctx " + g.QualifiedGoIdent(contextPackage.Ident("Context"))
	s += ", in *" + g.QualifiedGoIdent(method.Input.GoIdent)
	s += ") ("
	s += "*" + g.QualifiedGoIdent(method.Output.GoIdent)
	s += ", error)"

	return s
}

func genClientMethod(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, method *protogen.Method, index int) { //nolint:unparam
	service := method.Parent

	g.P("func (c *", unexport(service.GoName), "Client) ", clientSignature(g, method), "{")
	g.P("out := new(", method.Output.GoIdent, ")")
	g.P(`err := c.cc.Invoke(ctx, "`, method.Desc.Name(), `", in, out)`)
	g.P("if err != nil { return nil, err }")
	g.P("return out, nil")
	g.P("}")
	g.P()
}

func serverSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	var reqArgs []string
	reqArgs = append(reqArgs, g.QualifiedGoIdent(contextPackage.Ident("Context")))
	ret := "(*" + g.QualifiedGoIdent(method.Output.GoIdent) + ", error)"
	reqArgs = append(reqArgs, "*"+g.QualifiedGoIdent(method.Input.GoIdent))

	return method.GoName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

func genServerMethod(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, method *protogen.Method) string { //nolint:unparam
	service := method.Parent
	hname := fmt.Sprintf("_%s_%s_Handler", service.GoName, method.GoName)

	g.P("func ", hname, "(srv interface{}, ctx ", contextPackage.Ident("Context"), ", dec func(interface{}) error) (interface{}, error) {")
	g.P("in := new(", method.Input.GoIdent, ")")
	g.P("if err := dec(in); err != nil { return nil, err }")
	g.P("return srv.(", service.GoName, "Server).", method.GoName, "(ctx, in)")
	g.P("}")
	g.P()

	return hname
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }
