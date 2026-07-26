/**
 * 验证码提取：多信号打分，而非关键词硬匹配。
 *
 * 对文本中的每个候选 token（4-8 位数字，或含数字的 6-8 位字母数字）
 * 按上下文信号累计得分，取最高分且过阈值者：
 *  + 语境词邻近（验证/校验/code/OTP…），距离越近分越高
 *  + 冒号/是/为 等引导符紧跟其后
 *  + 【】/[] 强调包裹
 *  + 长度先验（6 位最常见）
 *  − 形似手机号/座机/年份/日期/金额/订单号
 *  − 与发件号码相同
 */

const CONTEXT_WORDS =
  /验证|校验|检验|动态|激活|确认|取件|提取|认证|驗證|檢驗|認證|verif|passcode|password|one[- ]?time|otp|code|kood|kode|codice|c[oó]digo|認証|コード|코드/gi;

const AMOUNT_NEAR = /[¥￥$€£]|元|块钱|金额|余额|人民币|dollars?|USD|CNY/i;
const ORDER_NEAR = /订单|单号|运单|快递单|流水号|order\s*(no|number|id)|tracking/i;
const PHONE_LIKE = /^(?:\+?86)?1[3-9]\d{9}$|^\+\d{7,15}$|^(?:400|800)\d{7}$/;
const DATE_LIKE =
  /^(?:19|20)\d{2}$|^(?:19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])$/;

type Candidate = {
  value: string;
  index: number;
  score: number;
};

export function extractOtp(text: string, sender?: string): string | null {
  if (!text || text.length > 1000) return null;

  // 语境词位置表（多信号之一，不再是硬性门槛）
  const contextHits: { start: number; end: number }[] = [];
  for (const m of text.matchAll(CONTEXT_WORDS)) {
    contextHits.push({ start: m.index, end: m.index + m[0].length });
  }

  const candidates: Candidate[] = [];

  // 候选 0：分组式验证码（983-497 / 123 456），整体作为候选，优先于拆分匹配
  const grouped = new Set<number>();
  for (const m of text.matchAll(
    /(?<![\d.\-])(\d{3}[- ]\d{3,4})(?!\d)(?!\.\d)(?!-\d)/g,
  )) {
    candidates.push({ value: m[1], index: m.index, score: 8 }); // 分组格式本身即验证码惯例，+8
    for (let i = m.index; i < m.index + m[1].length; i++) grouped.add(i);
  }

  // 候选 1：独立 4-8 位数字。
  // 排除：长数字串截段、真小数（数字.数字）、区间/电话分段（数字-数字）。
  // 保留：句尾标点（"916552."）与品牌前缀（"G-438277"）。
  for (const m of text.matchAll(
    /(?<![\d.])(?<!\d-)(\d{4,8})(?!\d)(?!\.\d)(?!-\d)/g,
  )) {
    if (grouped.has(m.index)) continue; // 已被分组候选覆盖
    candidates.push({ value: m[1], index: m.index, score: 0 });
  }
  // 候选 2：含数字的 6-8 位大写字母数字混合（如 A3B9K2）
  for (const m of text.matchAll(
    /(?<![A-Za-z0-9])(?=[A-Z0-9]*\d)([A-Z0-9]{6,8})(?![A-Za-z0-9])/g,
  )) {
    if (!/^\d+$/.test(m[1])) {
      candidates.push({ value: m[1], index: m.index, score: 0 });
    }
  }

  if (candidates.length === 0) return null;

  for (const c of candidates) {
    // —— 正向信号 ——

    // 语境词邻近：40 字符内线性衰减，最高 +50；
    // 语境词在候选之后（如 "183086 is your verification code"）打 0.8 折
    let bestCtx = 0;
    const cEnd = c.index + c.value.length;
    for (const hit of contextHits) {
      let dist: number;
      let factor = 1;
      if (hit.end <= c.index) {
        dist = c.index - hit.end;
      } else if (hit.start >= cEnd) {
        dist = hit.start - cEnd;
        factor = 0.8;
      } else {
        continue; // 重叠（候选包含在语境词里），不可能
      }
      if (dist < 40) {
        bestCtx = Math.max(bestCtx, 50 * factor * (1 - dist / 40));
      }
    }
    c.score += bestCtx;

    // 引导符：候选前 3 字符内出现 冒号/是/为/如下
    const lead = text.slice(Math.max(0, c.index - 4), c.index);
    if (/[:：]\s*$|[是为]\s*$/.test(lead)) c.score += 18;

    // 强调包裹：【123456】/ [123456] / “123456”
    const before = text[c.index - 1];
    const after = text[c.index + c.value.length];
    if (
      (before === "【" && after === "】") ||
      (before === "[" && after === "]") ||
      (before === "“" && after === "”") ||
      (before === '"' && after === '"')
    ) {
      c.score += 15;
    }

    // 长度先验：按数字位数计，6 位最常见，4 位次之
    const digitLen = c.value.replace(/\D/g, "").length;
    if (digitLen === 6) c.score += 12;
    else if (digitLen === 4) c.score += 6;
    else c.score += 3;

    // 纯数字（含分组分隔）比字母混合更常见
    if (/^[\d\- ]+$/.test(c.value)) c.score += 4;

    // —— 负向信号 ——

    const around = text.slice(
      Math.max(0, c.index - 12),
      c.index + c.value.length + 12,
    );

    if (PHONE_LIKE.test(c.value)) c.score -= 60;
    if (DATE_LIKE.test(c.value)) c.score -= 40;
    if (AMOUNT_NEAR.test(around)) c.score -= 30;
    if (ORDER_NEAR.test(around)) c.score -= 35;

    // 与发件号码相同/被其包含
    if (sender && (sender.includes(c.value) || c.value.includes(sender))) {
      c.score -= 60;
    }

    // 8 位纯数字且无语境支撑，多半是单号
    if (c.value.length >= 7 && /^\d+$/.test(c.value) && bestCtx === 0) {
      c.score -= 15;
    }
  }

  candidates.sort((a, b) => b.score - a.score);
  const best = candidates[0];

  // 阈值：必须有实质性的正向信号（仅长度先验不够）
  return best.score >= 30 ? best.value : null;
}

/** 复制文本，HTTP 环境降级到 execCommand */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {
    // fall through
  }
  try {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(ta);
    return ok;
  } catch {
    return false;
  }
}
