import React from "react";
import { render } from "@testing-library/react";

import MdmlabIcon from "./MdmlabIcon";

describe("MdmlabIcon - component", () => {
  it("renders", () => {
    const { container } = render(<MdmlabIcon name="success-check" />);
    expect(
      container.querySelector(".mdmlabicon-success-check")
    ).toBeInTheDocument();
  });
});
