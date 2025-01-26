import { Meta, StoryObj } from "@storybook/react";

import LastUpdatedNodeCount from "./LastUpdatedNodeCount";

const meta: Meta<typeof LastUpdatedNodeCount> = {
  title: "Components/LastUpdatedNodeCount",
  component: LastUpdatedNodeCount,
  args: {
    nodeCount: 40,
  },
};

export default meta;

type Story = StoryObj<typeof LastUpdatedNodeCount>;

export const Basic: Story = {};

export const WithLastUpdatedAt: Story = {
  args: {
    lastUpdatedAt: "2021-01-01T00:00:00Z",
  },
};
