import { Meta, StoryObj } from "@storybook/react";

import NodeStatusWebhookPreviewModal from "./NodeStatusWebhookPreviewModal";

const meta: Meta<typeof NodeStatusWebhookPreviewModal> = {
  title: "Components/NodeStatusWebhookPreviewModal",
  component: NodeStatusWebhookPreviewModal,
  args: { isTeamScope: false },
};

export default meta;

type Story = StoryObj<typeof NodeStatusWebhookPreviewModal>;

export const Basic: Story = {};
