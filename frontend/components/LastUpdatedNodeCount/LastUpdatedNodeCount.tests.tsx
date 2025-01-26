import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import LastUpdatedNodeCount from ".";

describe("Last updated node count", () => {
  it("renders node count and updated text", () => {
    const currentDate = new Date();
    currentDate.setDate(currentDate.getDate() - 2);
    const twoDaysAgo = currentDate.toISOString();

    render(<LastUpdatedNodeCount nodeCount={40} lastUpdatedAt={twoDaysAgo} />);

    const nodeCount = screen.getByText(/40/i);
    const updateText = screen.getByText("Updated 2 days ago");

    expect(nodeCount).toBeInTheDocument();
    expect(updateText).toBeInTheDocument();
  });
  it("renders never if missing timestamp", () => {
    render(<LastUpdatedNodeCount />);

    const text = screen.getByText("Updated never");

    expect(text).toBeInTheDocument();
  });

  it("renders tooltip on hover", async () => {
    render(<LastUpdatedNodeCount nodeCount={0} />);

    await fireEvent.mouseEnter(screen.getByText("Updated never"));

    expect(
      screen.getByText(/last time node data was updated/i)
    ).toBeInTheDocument();
  });
});
