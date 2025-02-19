import React from "react";

import { fireEvent, render, screen } from "@testing-library/react";
import paths from "router/paths";
import SummaryTile from "./SummaryTile";

const LOADING_OPACITY = 0.4;

describe("SummaryTile - component", () => {
  it("summary tile is hidden when showUI is false", () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI={false} // tested
        title="Windows nodes"
        iconName="windows"
        tooltip="Nodes on any Windows device"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const tile = screen.getByTestId("tile");

    expect(tile).not.toBeVisible();
  });

  it("renders loading state", () => {
    render(
      <SummaryTile
        count={200}
        isLoading // tested
        showUI
        title="Windows nodes"
        iconName="windows"
        tooltip="Nodes on any Windows device"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const tile = screen.getByTestId("tile");

    expect(tile).toHaveStyle(`opacity: ${LOADING_OPACITY}`);
    expect(tile).toBeVisible();
  });

  it("renders title, count, and image based on the information and data passed in", () => {
    render(
      <SummaryTile
        count={200} // tested
        isLoading={false}
        showUI
        title="Windows nodes" // tested
        iconName="windows" // tested
        tooltip="Nodes on any Windows device"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const title = screen.getByText("Windows nodes");
    const count = screen.getByText("200");
    const icon = screen.queryByTestId("windows-icon");

    expect(title).toBeInTheDocument();
    expect(count).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("does not render icon if not provided", () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI
        title="Windows nodes"
        iconName="windows"
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    const icon = screen.queryByRole("svg");

    expect(icon).toBeNull();
  });

  it("renders tooltip on title hover", async () => {
    render(
      <SummaryTile
        count={200}
        isLoading={false}
        showUI
        title="Windows nodes"
        iconName="windows"
        tooltip="Nodes on any Windows device" // tested
        path={paths.MANAGE_HOSTS_LABEL(10)}
      />
    );

    await fireEvent.mouseEnter(screen.getByText("Windows nodes"));

    expect(screen.getByText("Nodes on any Windows device")).toBeInTheDocument();
  });

  // Note: Cannot test path of react-router <Link/> without <Router/>
});
