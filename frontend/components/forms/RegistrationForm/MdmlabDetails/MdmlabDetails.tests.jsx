import React from "react";
import { render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import MdmlabDetails from "components/forms/RegistrationForm/MdmlabDetails";

describe("MdmlabDetails - form", () => {
  const handleSubmitSpy = jest.fn();
  it("renders", () => {
    render(<MdmlabDetails handleSubmit={handleSubmitSpy} />);

    expect(
      screen.getByRole("textbox", { name: "Mdmlab web address" })
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
  });

  it("validates the presence of the Mdmlab web address field", async () => {
    const { user } = renderWithSetup(
      <MdmlabDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Mdmlab web address must be completed")
    ).toBeInTheDocument();
  });

  it("validates the Mdmlab web address field starts with https://", async () => {
    const { user } = renderWithSetup(
      <MdmlabDetails handleSubmit={handleSubmitSpy} currentPage />
    );

    await user.type(
      screen.getByRole("textbox", { name: "Mdmlab web address" }),
      "http://gnar.Mdmlab.co"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(handleSubmitSpy).not.toHaveBeenCalled();
    expect(
      screen.getByText("Mdmlab web address must start with https://")
    ).toBeInTheDocument();
  });

  it("submits the form when valid", async () => {
    const { user } = renderWithSetup(
      <MdmlabDetails handleSubmit={handleSubmitSpy} currentPage />
    );
    // when
    await user.type(
      screen.getByRole("textbox", { name: "Mdmlab web address" }),
      "https://gnar.Mdmlab.co"
    );
    await user.click(screen.getByRole("button", { name: "Next" }));
    // then
    expect(handleSubmitSpy).toHaveBeenCalledWith({
      server_url: "https://gnar.Mdmlab.co",
    });
  });
});
