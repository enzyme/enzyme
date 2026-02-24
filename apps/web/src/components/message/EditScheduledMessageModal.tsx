import { useState, useMemo } from 'react';
import { getLocalTimeZone, today, Time, CalendarDate } from '@internationalized/date';
import type { DateValue, TimeValue } from 'react-aria-components';
import { Modal, Button, DatePicker, TimeField } from '../ui';
import { useUpdateScheduledMessage } from '../../hooks/useScheduledMessages';
import type { ScheduledMessage } from '@enzyme/api-client';

interface EditScheduledMessageModalProps {
  isOpen: boolean;
  onClose: () => void;
  message: ScheduledMessage | null;
}

function EditScheduledMessageForm({
  message,
  onClose,
}: {
  message: ScheduledMessage;
  onClose: () => void;
}) {
  const tz = getLocalTimeZone();
  const updateMutation = useUpdateScheduledMessage();

  const initialDate = new Date(message.scheduled_for as unknown as string);
  const [content, setContent] = useState(message.content);
  const [selectedDate, setSelectedDate] = useState<DateValue | null>(
    new CalendarDate(initialDate.getFullYear(), initialDate.getMonth() + 1, initialDate.getDate()),
  );
  const [selectedTime, setSelectedTime] = useState<TimeValue | null>(
    new Time(initialDate.getHours(), initialDate.getMinutes()),
  );

  const customDate = useMemo(() => {
    if (!selectedDate || !selectedTime) return null;
    return new Date(
      selectedDate.year,
      selectedDate.month - 1,
      selectedDate.day,
      selectedTime.hour,
      selectedTime.minute,
    );
  }, [selectedDate, selectedTime]);

  const isValid = customDate !== null && customDate > new Date() && content.trim() !== '';

  const handleSave = () => {
    if (!isValid || !customDate) return;

    updateMutation.mutate(
      {
        id: message.id,
        input: {
          content: content.trim(),
          scheduled_for: customDate.toISOString(),
        },
      },
      {
        onSuccess: () => onClose(),
      },
    );
  };

  const tzAbbr = useMemo(() => {
    return (
      new Intl.DateTimeFormat('en-US', { timeZoneName: 'short' })
        .formatToParts(new Date())
        .find((p) => p.type === 'timeZoneName')?.value || tz
    );
  }, [tz]);

  return (
    <div className="space-y-4">
      <div>
        <label className="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Message
        </label>
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          rows={3}
          className="w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100"
        />
      </div>

      <div className="flex items-end gap-3">
        <div className="flex-1">
          <DatePicker
            label="Date"
            value={selectedDate}
            onChange={setSelectedDate}
            minValue={today(tz)}
          />
        </div>
        <div className="flex-1">
          <TimeField label="Time" value={selectedTime} onChange={setSelectedTime} />
        </div>
        <span className="pb-1 text-xs text-gray-400 dark:text-gray-500">{tzAbbr}</span>
      </div>

      <div className="flex justify-end gap-2">
        <Button variant="secondary" onPress={onClose}>
          Cancel
        </Button>
        <Button onPress={handleSave} isDisabled={!isValid} isLoading={updateMutation.isPending}>
          Update
        </Button>
      </div>
    </div>
  );
}

export function EditScheduledMessageModal({
  isOpen,
  onClose,
  message,
}: EditScheduledMessageModalProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Edit scheduled message">
      {message && <EditScheduledMessageForm key={message.id} message={message} onClose={onClose} />}
    </Modal>
  );
}
