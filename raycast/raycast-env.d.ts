/// <reference types="@raycast/api">

/* 🚧 🚧 🚧
 * This file is auto-generated from the extension's manifest.
 * Do not modify manually. Instead, update the `package.json` file.
 * 🚧 🚧 🚧 */

/* eslint-disable @typescript-eslint/ban-types */

type ExtensionPreferences = {
  /** mux Binary Path - Absolute path to the mux binary. Leave blank to auto-detect (~/go/bin, Homebrew, /usr/local/bin). */
  "muxPath"?: string
}

/** Preferences accessible in all the extension's commands */
declare type Preferences = ExtensionPreferences

declare namespace Preferences {
  /** Preferences accessible in the `apply-preset` command */
  export type ApplyPreset = ExtensionPreferences & {}
}

declare namespace Arguments {
  /** Arguments passed to the `apply-preset` command */
  export type ApplyPreset = {}
}

