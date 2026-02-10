import { describe, it, expect, vi } from 'vitest';
import { render, screen, userEvent } from '../../test-utils';
import { RadioGroup, Radio } from './RadioGroup';

describe('RadioGroup', () => {
  it('renders radio options', () => {
    render(
      <RadioGroup label="Choose option">
        <Radio value="a">Option A</Radio>
        <Radio value="b">Option B</Radio>
      </RadioGroup>,
    );

    expect(screen.getByText('Option A')).toBeInTheDocument();
    expect(screen.getByText('Option B')).toBeInTheDocument();
  });

  it('renders label when provided', () => {
    render(
      <RadioGroup label="Select preference">
        <Radio value="1">One</Radio>
        <Radio value="2">Two</Radio>
      </RadioGroup>,
    );

    expect(screen.getByText('Select preference')).toBeInTheDocument();
  });

  it('does not render label when not provided', () => {
    render(
      <RadioGroup>
        <Radio value="1">One</Radio>
        <Radio value="2">Two</Radio>
      </RadioGroup>,
    );

    // The label element should not be present
    const radiogroup = screen.getByRole('radiogroup');
    expect(radiogroup).toBeInTheDocument();
  });

  it('selects option on click', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(
      <RadioGroup onChange={handleChange}>
        <Radio value="a">Option A</Radio>
        <Radio value="b">Option B</Radio>
      </RadioGroup>,
    );

    const radios = screen.getAllByRole('radio');
    await user.click(radios[1]);

    expect(handleChange).toHaveBeenCalledWith('b');
  });

  it('supports controlled value', () => {
    render(
      <RadioGroup value="b">
        <Radio value="a">Option A</Radio>
        <Radio value="b">Option B</Radio>
      </RadioGroup>,
    );

    const radios = screen.getAllByRole('radio');
    expect(radios[0]).not.toBeChecked();
    expect(radios[1]).toBeChecked();
  });

  it('supports disabled state on group', () => {
    render(
      <RadioGroup isDisabled>
        <Radio value="a">Option A</Radio>
        <Radio value="b">Option B</Radio>
      </RadioGroup>,
    );

    const radios = screen.getAllByRole('radio');
    expect(radios[0]).toBeDisabled();
    expect(radios[1]).toBeDisabled();
  });

  it('supports disabled state on individual radio', () => {
    render(
      <RadioGroup>
        <Radio value="a">Option A</Radio>
        <Radio value="b" isDisabled>
          Option B
        </Radio>
      </RadioGroup>,
    );

    const radios = screen.getAllByRole('radio');
    expect(radios[0]).not.toBeDisabled();
    expect(radios[1]).toBeDisabled();
  });

  it('applies data-selected attribute to selected radio', async () => {
    const user = userEvent.setup();

    const { container } = render(
      <RadioGroup>
        <Radio value="a">Option A</Radio>
        <Radio value="b">Option B</Radio>
      </RadioGroup>,
    );

    const radios = screen.getAllByRole('radio');
    await user.click(radios[0]);

    // React Aria sets data-selected on the AriaRadio wrapper element
    // which is where Tailwind data-[selected]: styles are applied
    const selectedElements = container.querySelectorAll('[data-selected]');
    expect(selectedElements.length).toBe(1);

    // The selected element should be associated with radio "a"
    const selectedRadio = selectedElements[0].querySelector('input[type="radio"]');
    expect(selectedRadio).toBe(radios[0]);
  });

  it('supports keyboard navigation with arrow keys', async () => {
    const user = userEvent.setup();
    const handleChange = vi.fn();

    render(
      <RadioGroup onChange={handleChange}>
        <Radio value="a">Option A</Radio>
        <Radio value="b">Option B</Radio>
        <Radio value="c">Option C</Radio>
      </RadioGroup>,
    );

    const radios = screen.getAllByRole('radio');

    // Focus first radio
    radios[0].focus();
    expect(radios[0]).toHaveFocus();

    // Navigate down with arrow key
    await user.keyboard('{ArrowDown}');
    expect(radios[1]).toHaveFocus();
    expect(handleChange).toHaveBeenCalledWith('b');
  });
});
