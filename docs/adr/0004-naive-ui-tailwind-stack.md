# NaiveUI + Tailwind as the web UI component stack

The web UI uses NaiveUI as the component library and Tailwind CSS for utility styling. NaiveUI provides a complete dark-theme-capable Vue 3 component set (data tables, layout, notifications, buttons) that matches the aesthetic of a server management tool. Tailwind fills the gaps where NaiveUI's inline props are insufficient without requiring a separate design system to maintain.

The alternative was a headless component library (e.g. Radix Vue) paired with pure Tailwind — rejected because the effort to build accessible, consistent components from scratch isn't justified for a single-operator admin UI. Vuetify and PrimeVue were not considered; NaiveUI was already installed and tested working.
