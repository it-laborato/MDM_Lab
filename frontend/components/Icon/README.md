# mdmlab icons library

## Migrating icons

There is a current migration of icons from pixelated PNG files to SVG components.

For internal developers, the source of truth for icon SVGs is on [mdmlab style guide](https://www.figma.com/file/qbjRu8jf01BzEfdcge1dgu/mdmlab-style-guide-2022-(WIP)?type=design&node-id=213-30309&t=kZelMf1i2hQ7GAaI-0) on Figma. Current progress on migrating icons is tracked on a [mdmlab icons spreadsheet](https://docs.google.com/spreadsheets/d/1dNcppmEmnlDvozNKQZ7fgZlqkf7_delkDMtIJCJ2X20/edit?usp=sharing).

## Code organization

### Icon directory

The `Icon` directory includes global styling, the global `Icon` component, and the storybook integration.

### icons directory

The `icons` directory includes separate files for all SVG components for mdmlab icons.

Each component can be modified to take various props including but not limited to `color` using `frontend/styles/var/colors.ts` or `size` using `frontend/styles/var/icon_sizes`.

`index.ts` maps all icon names used in `<Icon name="icon-name" />` to its respective SVG component.

## How to view mdmlab icons

Figma extensions for SVG previews cannot render all SVG components. Use [Storybook](../../README.md#storybook) to easily view mdmlab icons by running `yarn storybook`.

