package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"github.com/olekukonko/tablewriter"
	"github.com/rogerwelin/cfnctl/pkg/client"
)

type stackResourceEvents struct {
	events []types.StackResource
}

func tableOutputter(events []types.StackResource, writer io.Writer) {
	tableData := [][]string{}
	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Logical ID", "Physical ID", "Type", "Status", "Status Reason"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.Bold},
	)

	if len(events) == 0 {
		return
	}

	for _, item := range events {
		var physicalID string
		var statusReason string

		if item.PhysicalResourceId != nil {
			physicalID = *item.PhysicalResourceId
		} else {
			physicalID = "-"
		}

		if item.ResourceStatusReason != nil {
			statusReason = *item.ResourceStatusReason
		} else {
			statusReason = "-"
		}

		arr := []string{
			*item.LogicalResourceId,
			physicalID,
			*item.ResourceType,
			string(item.ResourceStatus),
			statusReason,
		}
		tableData = append(tableData, arr)
	}

	for i := range tableData {
		switch tableData[i][3] {
		case "CREATE_COMPLETE":
			table.Rich(tableData[i], []tablewriter.Colors{{}, {}, {}, {tablewriter.Normal, tablewriter.FgHiGreenColor}, {}})
		case "CREATE_IN_PROGRESS":
			table.Rich(tableData[i], []tablewriter.Colors{{}, {}, {}, {tablewriter.Normal, tablewriter.FgHiBlueColor}, {}})
		case "CREATE_FAILED":
			table.Rich(tableData[i], []tablewriter.Colors{{tablewriter.Normal, tablewriter.FgHiRedColor}})
		default:
			table.Append(tableData[i])
		}
	}

	table.Render()
}

func streamStackResources(ch <-chan stackResourceEvents, done <-chan bool) {
	writer := uilive.New()
	writer.Start()

	for {
		select {
		case <-done:
			fmt.Println("we've ended")
			writer.Stop()
			return
		case item := <-ch:
			tableOutputter(item.events, writer)
		}
	}
}

func cfnCtlApply(ctl *client.Cfnctl) error {

	greenBold := color.New(color.Bold, color.FgHiGreen).SprintFunc()
	whiteBold := color.New(color.Bold).SprintfFunc()

	eventsChan := make(chan stackResourceEvents)
	doneChan := make(chan bool)

	pc, err := cfnCtlPlan(ctl)
	if err != nil {
		panic(err)
	}

	if !pc.containsChanges {
		fmt.Printf("\n%s. %s\n\n", greenBold("No changes"), whiteBold("Your infrastructure matches the configuration"))
		fmt.Print("Cfnctl has compared your real infrastructure against your configuration and found no differences, so no changes are needed.\n")
		fmt.Printf("\n%s %d added, %d changed, %d destroyed\n", greenBold("Apply complete! Resources:"), (pc.changes["add"]), pc.changes["change"], pc.changes["destroy"])
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s\n"+
		"  Cfnctl will perform the actions described above.\n"+
		"  Only 'yes' will be accepted to approve.\n\n"+
		"  %s", whiteBold("Do you want to perform the following actions?"), whiteBold("Enter a value: "))

	choice, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	choice = strings.TrimSuffix(choice, "\n")

	if choice != "yes" {
		fmt.Println("\nApply cancelled.")
		return nil
	}

	err = ctl.ApplyChangeSet("dynamolambda")
	if err != nil {
		return err
	}

	go streamStackResources(eventsChan, doneChan)

	for {
		time.Sleep(500 * time.Millisecond)
		status, err := ctl.DescribeStack("dynamolambda")
		if err != nil {
			return err
		}
		if status == "UPDATE_COMPLETE" || status == "CREATE_FAILED" || status == "CREATE_COMPLETE" {
			break
		} else {
			event, err := ctl.DescribeStackResources("dynamolambda")
			if err != nil {
				return err
			}
			eventsChan <- stackResourceEvents{events: event}

		}

	}

	close(eventsChan)
	doneChan <- true

	return nil
}
