import type { SVGProps } from "react";

type IconProps = SVGProps<SVGSVGElement> & { size?: number };

function iconProps({ size = 20, className, ...rest }: IconProps) {
  return {
    width: size,
    height: size,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.8,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    className,
    "aria-hidden": true as const,
    ...rest,
  };
}

/** Monochrome pigeon mark — black/white only */
export function LogoMark({ size = 28, className, ...rest }: IconProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      className={className}
      aria-hidden
      {...rest}
    >
      <rect width="32" height="32" rx="9" fill="currentColor" />
      <path
        d="M8.8 19.2c1.1-3.6 3.9-5.9 7.8-6.3 1.5-.2 2.9.1 4.1.8"
        fill="none"
        stroke="var(--bg)"
        strokeWidth="1.7"
        strokeLinecap="round"
      />
      <path
        d="M17.8 11.1c.9-1.5 2.3-2.5 4.1-2.8"
        fill="none"
        stroke="var(--bg)"
        strokeWidth="1.7"
        strokeLinecap="round"
        opacity="0.85"
      />
      <path
        d="M11.4 14.7c-.1 2.3.5 4.4 2.1 6.1 1 1.1 2.4 1.9 4 2.2"
        fill="none"
        stroke="var(--bg)"
        strokeWidth="1.7"
        strokeLinecap="round"
      />
      <circle cx="21" cy="14.5" r="1.3" fill="var(--bg)" />
      <path
        d="M22.4 14.1c1 .4 1.9.2 2.7-.6"
        fill="none"
        stroke="var(--bg)"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
    </svg>
  );
}

export function IconSearch(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <circle cx="11" cy="11" r="6.5" />
      <path d="M20 20l-3.2-3.2" />
    </svg>
  );
}

export function IconSettings(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <circle cx="12" cy="12" r="3" />
      <path d="M12 3.5v2M12 18.5v2M4.9 6.5l1.4 1.4M17.7 17.7l1.4 1.4M3.5 12h2M18.5 12h2M4.9 17.5l1.4-1.4M17.7 6.3l1.4-1.4" />
    </svg>
  );
}

export function IconSun(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <circle cx="12" cy="12" r="3.75" />
      <path d="M12 3.5v1.6M12 18.9v1.6M4.9 4.9l1.1 1.1M18 18l1.1 1.1M3.5 12h1.6M18.9 12h1.6M4.9 19.1l1.1-1.1M18 6l1.1-1.1" />
    </svg>
  );
}

export function IconMoon(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M19.2 14.1A7.2 7.2 0 1 1 9.9 4.8 5.8 5.8 0 0 0 19.2 14.1z" />
    </svg>
  );
}

export function IconMonitor(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <rect x="3.5" y="4.5" width="17" height="12" rx="2" />
      <path d="M8 20h8M12 16.5V20" />
    </svg>
  );
}

export function IconLogout(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M10 7.5V5.8A1.8 1.8 0 0 1 11.8 4h6.4A1.8 1.8 0 0 1 20 5.8v12.4a1.8 1.8 0 0 1-1.8 1.8h-6.4A1.8 1.8 0 0 1 10 18.2v-1.7" />
      <path d="M15 12H4m0 0 2.8-2.8M4 12l2.8 2.8" />
    </svg>
  );
}

export function IconClose(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M6.5 6.5l11 11M17.5 6.5l-11 11" />
    </svg>
  );
}

export function IconTrash(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M4.5 7h15M9.5 7V5.4A1.4 1.4 0 0 1 10.9 4h2.2a1.4 1.4 0 0 1 1.4 1.4V7m-7 0 .9 11.2A1.8 1.8 0 0 0 10 20h4a1.8 1.8 0 0 0 1.8-1.8L16.7 7" />
    </svg>
  );
}

export function IconInbox(props: IconProps) {
  return (
    <svg {...iconProps({ ...props, size: props.size ?? 20 })}>
      <path d="M4.5 13h3.6l1.8 2.6h4.2L16 13h3.5" />
      <path d="M4.5 13V7.5A2 2 0 0 1 6.5 5.5h11A2 2 0 0 1 19.5 7.5v9A2 2 0 0 1 17.5 18.5h-11A2 2 0 0 1 4.5 16.5V13z" />
    </svg>
  );
}

export function IconInboxLarge(props: IconProps) {
  return (
    <svg {...iconProps({ ...props, size: props.size ?? 36 })}>
      <path d="M4.5 13h3.6l1.8 2.6h4.2L16 13h3.5" />
      <path d="M4.5 13V7.5A2 2 0 0 1 6.5 5.5h11A2 2 0 0 1 19.5 7.5v9A2 2 0 0 1 17.5 18.5h-11A2 2 0 0 1 4.5 16.5V13z" />
    </svg>
  );
}

export function IconAlert(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M12 9v4.2M12 16.8h.01" />
      <path d="M10.4 4.6 2.9 17.8A1.8 1.8 0 0 0 4.5 20.5h15a1.8 1.8 0 0 0 1.6-2.7L13.6 4.6a1.8 1.8 0 0 0-3.2 0z" />
    </svg>
  );
}

export function IconDevice(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <rect x="7" y="3.5" width="10" height="17" rx="2" />
      <path d="M11 17h2" />
    </svg>
  );
}

export function IconShield(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M12 3.5 5.5 6.2v5.5c0 4.1 2.7 6.9 6.5 8.3 3.8-1.4 6.5-4.2 6.5-8.3V6.2L12 3.5z" />
    </svg>
  );
}

export function IconSend(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M20.5 4 10.2 14.2" />
      <path d="M20.5 4 14.2 20.5l-3.5-6.3-6.2-3.5L20.5 4z" />
    </svg>
  );
}

/** Bell — used for push/notification channels nav */
export function IconBell(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M6.2 16.5h11.6c.9 0 1.4-1 .9-1.7l-1.2-1.7V11a5.5 5.5 0 1 0-11 0v2.1l-1.2 1.7c-.5.7 0 1.7.9 1.7z" />
      <path d="M10 16.5a2 2 0 0 0 4 0" />
    </svg>
  );
}

export function IconSignalBars(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M5 16.5v-2M9.5 16.5v-5M14 16.5V8M18.5 16.5V5.5" />
    </svg>
  );
}

export function IconPlus(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M12 5.5v13M5.5 12h13" />
    </svg>
  );
}

export function IconCheck(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M5.5 12.5 10 17l8.5-9.5" />
    </svg>
  );
}

export function IconEye(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M2.8 12S6.2 6.5 12 6.5 21.2 12 21.2 12 17.8 17.5 12 17.5 2.8 12 2.8 12z" />
      <circle cx="12" cy="12" r="2.4" />
    </svg>
  );
}

export function IconEyeOff(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M3.5 3.5l17 17" />
      <path d="M10.7 10.7A2.4 2.4 0 0 0 12 14.4c.5 0 .9-.1 1.3-.4" />
      <path d="M7.2 7.5C5.3 8.7 3.9 10.5 3 12c0 0 3.4 5.5 9 5.5 1.6 0 3-.4 4.2-1" />
      <path d="M14 9.3c1.1.7 2 1.7 2.7 2.7 0 0-1.1 1.9-3 3.1" />
      <path d="M9.9 6.6c.7-.1 1.4-.1 2.1-.1 5.6 0 9 5.5 9 5.5a15 15 0 0 1-1.6 2.1" />
    </svg>
  );
}

export function IconChevron(props: IconProps) {
  return (
    <svg {...iconProps(props)}>
      <path d="M9 6l6 6-6 6" />
    </svg>
  );
}
