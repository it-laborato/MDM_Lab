import React from "react";

import { render, screen } from "@testing-library/react";

import TeamNodeExpiryToggle from "./TeamNodeExpiryToggle";

describe("TeamNodeExpiryToggle component", () => {
  // global setting disabled
  it("Renders correctly with no global window set", () => {
    render(
      <TeamNodeExpiryToggle
        globalNodeExpiryEnabled={false}
        globalNodeExpiryWindow={undefined}
        teamExpiryEnabled={false}
        setTeamExpiryEnabled={jest.fn()}
      />
    );

    expect(screen.getByText(/Enable node expiry/)).toBeInTheDocument();
    expect(screen.queryByText(/Node expiry is globally enabled/)).toBeNull();
  });

  // global setting enabled
  it("Renders as expected when global enabled, local disabled", () => {
    render(
      <TeamNodeExpiryToggle
        globalNodeExpiryEnabled
        globalNodeExpiryWindow={2}
        teamExpiryEnabled={false}
        setTeamExpiryEnabled={jest.fn()}
      />
    );

    expect(screen.getByText(/Enable node expiry/)).toBeInTheDocument();
    expect(
      screen.getByText(/Node expiry is globally enabled/)
    ).toBeInTheDocument();
    expect(screen.getByText(/Add custom expiry window/)).toBeInTheDocument();
  });

  it("Renders as expected when global enabled, local enabled", () => {
    render(
      <TeamNodeExpiryToggle
        globalNodeExpiryEnabled
        globalNodeExpiryWindow={2}
        teamExpiryEnabled
        setTeamExpiryEnabled={jest.fn()}
      />
    );

    expect(screen.getByText(/Enable node expiry/)).toBeInTheDocument();
    expect(
      screen.getByText(/Node expiry is globally enabled/)
    ).toBeInTheDocument();
    expect(screen.queryByText(/Add custom expiry window/)).toBeNull();
  });
});
