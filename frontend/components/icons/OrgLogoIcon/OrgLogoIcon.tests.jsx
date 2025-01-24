import React from "react";
import { render, screen } from "@testing-library/react";

import mdmlabAvatar from "../../../../assets/images/mdmlab-avatar-24x24@2x.png";
import OrgLogoIcon from "./OrgLogoIcon";

describe("OrgLogoIcon - component", () => {
  it("renders the mdmlab Logo by default", () => {
    render(<OrgLogoIcon />);

    // expect(component.state("imageSrc")).toEqual(mdmlabAvatar);
    expect(screen.getByRole("img")).toHaveAttribute("src", mdmlabAvatar);
  });

  it("renders the image source when it is valid", () => {
    render(<OrgLogoIcon src="/assets/images/avatar.svg" />);

    expect(screen.getByRole("img")).toHaveAttribute(
      "src",
      "/assets/images/avatar.svg"
    );
  });
});
