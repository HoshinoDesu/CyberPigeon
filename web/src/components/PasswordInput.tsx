"use client";

import { useState } from "react";
import { IconEye, IconEyeOff } from "./icons";

type Props = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  autoComplete?: string;
  name?: string;
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
};

export function PasswordInput({
  value,
  onChange,
  placeholder,
  disabled,
  autoComplete,
  name,
  onKeyDown,
}: Props) {
  const [show, setShow] = useState(false);

  return (
    <div className="relative">
      <input
        type={show ? "text" : "password"}
        className="field"
        style={{ paddingRight: "2.75rem" }}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        autoComplete={autoComplete}
        name={name}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
      />
      <button
        type="button"
        className="pressable absolute top-1/2 right-2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full text-[var(--label-secondary)]"
        onClick={() => setShow((v) => !v)}
        tabIndex={-1}
        aria-label={show ? "隐藏密码" : "显示密码"}
      >
        {show ? <IconEye size={18} /> : <IconEyeOff size={18} />}
      </button>
    </div>
  );
}
