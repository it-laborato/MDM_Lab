import { Meta, StoryObj } from "@storybook/react";

import MdmlabMarkdown from "./MdmlabMarkdown";

const TestMarkdown = `
# Test Markdown

## This is a heading

### This is a subheading

#### This is a subsubheading


---
**bold**

*italic*

[test link](https://www.fleetdm.com)

- test list item 1
- test list item 2
- test list item 3

> test blockquote

\`code text\`
`;

const meta: Meta<typeof MdmlabMarkdown> = {
  title: "Components/MdmlabMarkdown",
  component: MdmlabMarkdown,
  args: { markdown: TestMarkdown },
};

export default meta;

type Story = StoryObj<typeof MdmlabMarkdown>;

export const Basic: Story = {};
