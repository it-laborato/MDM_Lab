import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { noop } from "lodash";

import { createMockLabel } from "__mocks__/labelsMock";
import createMockNode from "__mocks__/nodeMock";
import TargetOption from "./TargetOption";

describe("TargetOption - component", () => {
  const onMoreInfoClickSpy = jest.fn();
  const onMoreInfoClick = () => {
    return onMoreInfoClickSpy;
  };
  it("renders a label option for label targets", () => {
    const { container } = render(
      <TargetOption
        onSelect={noop}
        onMoreInfoClick={onMoreInfoClick}
        target={createMockLabel({
          target_type: "labels",
          count: 20,
        })}
      />
    );
    expect(container.querySelectorAll(".is-label").length).toEqual(1);
    expect(screen.getByText(`20 nodes`)).toBeInTheDocument();
  });

  it("renders a node option for node targets", () => {
    const { container } = render(
      <TargetOption
        onSelect={noop}
        onMoreInfoClick={onMoreInfoClick}
        target={createMockNode({ target_type: "nodes", platform: "windows" })}
      />
    );
    expect(container.querySelectorAll(".is-node").length).toEqual(1);
    expect(container.querySelectorAll("i.mdmlabicon-windows").length).toEqual(1);
    expect(screen.getByText(createMockNode().primary_ip)).toBeInTheDocument();
  });

  it("calls the onSelect prop when + icon button is clicked", () => {
    const onSelectSpy = jest.fn();
    const { container } = render(
      <TargetOption
        onMoreInfoClick={onMoreInfoClick}
        onSelect={onSelectSpy}
        target={createMockNode()}
      />
    );

    const addButton = container.querySelector(".target-option__add-btn");

    expect(addButton).toBeInTheDocument();

    if (addButton) {
      fireEvent.click(addButton);
      expect(onSelectSpy).toHaveBeenCalled();
    }
  });

  it("calls the onMoreInfoClick prop when the item content is clicked", () => {
    const { container } = render(
      <TargetOption
        onSelect={noop}
        onMoreInfoClick={onMoreInfoClick}
        target={createMockNode()}
      />
    );

    const moreInfo = container.querySelector(".target-option__target-content");

    expect(moreInfo).toBeInTheDocument();

    if (moreInfo) {
      fireEvent.click(moreInfo);
      expect(onMoreInfoClickSpy).toHaveBeenCalled();
    }
  });
});
