package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/docopt/docopt-go"
	"github.com/proxypoke/i3ipc"
	"github.com/reconquest/karma-go"
)

var (
	version = "[manual build]"
	usage   = "i3re " + version + `

Resize i3 current focused window with pixel settings.

Usage:
  i3re [options] -w <px> -h <px>
  i3re [options] -w <px>
  i3re [options] -h <px>
  i3re -h | --help
  i3re --version

Options:
  -w --width <px>   Resize window to specified width.
  -h --height <px> Resize window to specified height.  
  --help            Show this screen.
  --version         Show version.
`
)

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		panic(err)
	}

	i3, err := i3ipc.GetIPCSocket()
	if err != nil {
		log.Fatal(err)
	}

	defer i3.Close()

	var width int
	if raw, ok := args["--width"].(string); ok {
		width, err = strconv.Atoi(raw)
		if err != nil {
			log.Fatal(err)
		}
	}

	var height int
	if raw, ok := args["--height"].(string); ok {
		height, err = strconv.Atoi(raw)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = resize(i3, width, height)
	if err != nil {
		log.Fatal(err)
	}
}

func resize(
	i3 *i3ipc.IPCSocket,
	width int,
	height int,
) error {
	workspace, err := getFocusedWorkspace(i3)
	if err != nil {
		return err
	}

	window, err := getFocusedWindow(i3)
	if err != nil {
		return err
	}

	var (
		workspaceWidth  = float64(workspace.Rect.Width)
		workspaceHeight = float64(workspace.Rect.Height)
	)

	var (
		windowWidth  = float64(window.Window_Rect.Width)
		windowheight = float64(window.Window_Rect.Height)
	)

	if width > 0 {
		command := getResizeCommand("width", workspaceWidth, windowWidth, float64(width))

		_, err := i3.Command(command)
		if err != nil {
			return karma.Describe("command", command).Format(
				err,
				"unable to resize window",
			)
		}
	}

	if height > 0 {
		command := getResizeCommand("height", workspaceHeight, windowheight, float64(height))

		_, err := i3.Command(command)
		if err != nil {
			return karma.Describe("command", command).Format(
				err,
				"unable to resize window",
			)
		}
	}

	return nil
}

func getResizeCommand(
	direction string,
	workspaceValue float64,
	windowValue float64,
	needValue float64,
) string {
	windowPercent := 100 * windowValue / workspaceValue
	needPercent := 100 * float64(needValue) / float64(workspaceValue)

	ppt := needPercent - windowPercent

	op := "grow"
	if ppt < 0 {
		op = "shrink"
		ppt = ppt * -1
	}

	command := fmt.Sprintf(
		"resize %s %s 1 px or %d ppt",
		op, direction, int64(ppt),
	)

	return command
}

func getFocusedWorkspace(i3 *i3ipc.IPCSocket) (i3ipc.Workspace, error) {
	workspaces, err := i3.GetWorkspaces()
	if err != nil {
		return i3ipc.Workspace{}, err
	}

	for _, workspace := range workspaces {
		if workspace.Focused {
			return workspace, nil
		}
	}

	return i3ipc.Workspace{}, fmt.Errorf("could not found focused workspace")
}

func getFocusedWindow(i3 *i3ipc.IPCSocket) (i3ipc.I3Node, error) {
	tree, err := i3.GetTree()
	if err != nil {
		return tree, err
	}

	var walker func(i3ipc.I3Node) (i3ipc.I3Node, bool)

	walker = func(node i3ipc.I3Node) (i3ipc.I3Node, bool) {
		for _, subnode := range node.Nodes {
			if subnode.Focused {
				subnode.Layout = node.Layout
				return subnode, true
			}

			activeNode, ok := walker(subnode)
			if ok {
				return activeNode, true
			}
		}

		return node, false
	}

	node, _ := walker(tree)

	return node, nil
}
