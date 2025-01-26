import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import createMockNode from "__mocks__/nodeMock";
import { INode } from "interfaces/node";

import TargetsInput from "./TargetsInput";
import { ITargestInputNodeTableConfig } from "./TargetsInputNodesTableConfig";

describe("TargetsInput", () => {
  it("renders the search table based on the custom configuration passed in", () => {
    const testNodes: INode[] = [
      createMockNode({
        display_name: "testNode",
        public_ip: "123.456.789.0",
        computer_name: "testName",
      }),
    ];

    const testTableConfig: ITargestInputNodeTableConfig[] = [
      {
        Header: "Name",
        accessor: "display_name",
      },
      {
        Header: "IP Address",
        accessor: "public_ip",
      },
    ];

    render(
      <TargetsInput
        searchText="test"
        searchResults={testNodes}
        isTargetsLoading={false}
        hasFetchError={false}
        targetedNodes={[]}
        searchResultsTableConfig={testTableConfig}
        selectedNodesTableConifg={[]}
        setSearchText={noop}
        handleRowSelect={noop}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("IP Address")).toBeInTheDocument();
    expect(screen.getByText("testNode")).toBeInTheDocument();
    expect(screen.getByText("123.456.789.0")).toBeInTheDocument();
    expect(screen.queryByText("testName")).not.toBeInTheDocument();
  });

  it("renders the results table based on the custom configuration passed in", () => {
    const testNodes: INode[] = [
      createMockNode({
        display_name: "testNode",
        public_ip: "123.456.789.0",
        computer_name: "testName",
      }),
    ];

    const testTableConfig: ITargestInputNodeTableConfig[] = [
      {
        Header: "Name",
        accessor: "display_name",
      },
      {
        Header: "IP Address",
        accessor: "public_ip",
      },
    ];

    render(
      <TargetsInput
        searchText=""
        searchResults={[]}
        isTargetsLoading={false}
        hasFetchError={false}
        targetedNodes={testNodes}
        searchResultsTableConfig={[]}
        selectedNodesTableConifg={testTableConfig}
        setSearchText={noop}
        handleRowSelect={noop}
      />
    );

    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("IP Address")).toBeInTheDocument();
    expect(screen.getByText("testNode")).toBeInTheDocument();
    expect(screen.getByText("123.456.789.0")).toBeInTheDocument();
    expect(screen.queryByText("testName")).not.toBeInTheDocument();
  });
});
