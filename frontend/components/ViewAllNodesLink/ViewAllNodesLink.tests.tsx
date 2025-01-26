import React from "react";
import { render, screen } from "@testing-library/react";
import ViewAllNodesLink from "./ViewAllNodesLink";

describe("ViewAllNodesLink - component", () => {
  it("renders View all nodes text and icon", () => {
    render(<ViewAllNodesLink />);

    const text = screen.getByText("View all nodes");
    const icon = screen.getByTestId("chevron-right-icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
  });

  it("hides text when set to condensed ", async () => {
    render(<ViewAllNodesLink queryParams={{ status: "online" }} condensed />);
    const text = screen.queryByText("View all nodes");

    expect(text).toBeNull();
  });
});
