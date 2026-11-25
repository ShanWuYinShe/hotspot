import { describe, expect, it } from 'vitest';
import { resolveRovingRadioIndex } from './roving-radio';

describe('resolveRovingRadioIndex', () => {
	it('方向键在组内循环移动', () => {
		expect(resolveRovingRadioIndex(0, 3, 'ArrowRight')).toBe(1);
		expect(resolveRovingRadioIndex(1, 3, 'ArrowDown')).toBe(2);
		expect(resolveRovingRadioIndex(2, 3, 'ArrowRight')).toBe(0);
		expect(resolveRovingRadioIndex(0, 3, 'ArrowLeft')).toBe(2);
		expect(resolveRovingRadioIndex(1, 3, 'ArrowUp')).toBe(0);
	});

	it('Home/End 跳到首末项', () => {
		expect(resolveRovingRadioIndex(2, 3, 'Home')).toBe(0);
		expect(resolveRovingRadioIndex(0, 3, 'End')).toBe(2);
	});

	it('非方向键返回 null，不劫持普通输入', () => {
		expect(resolveRovingRadioIndex(0, 3, 'Enter')).toBeNull();
		expect(resolveRovingRadioIndex(0, 3, 'Tab')).toBeNull();
		expect(resolveRovingRadioIndex(0, 3, 'a')).toBeNull();
	});

	it('越界下标被夹紧且空组安全', () => {
		expect(resolveRovingRadioIndex(9, 3, 'ArrowLeft')).toBe(1);
		expect(resolveRovingRadioIndex(-2, 3, 'ArrowRight')).toBe(1);
		expect(resolveRovingRadioIndex(0, 0, 'ArrowRight')).toBeNull();
	});
});
