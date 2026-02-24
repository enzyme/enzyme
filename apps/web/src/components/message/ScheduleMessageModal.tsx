import { useState, useMemo } from 'react';
import { today, getLocalTimeZone, Time } from '@internationalized/date';
import type { DateValue, TimeValue } from 'react-aria-components';
import { Modal, Button, DatePicker, TimeField } from '../ui';

interface ScheduleMessageModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSchedule: (scheduledFor: string) => void;
}

function formatScheduledDate(date: Date): string {
  return (
    date.toLocaleDateString('en-US', {
      weekday: 'long',
      month: 'short',
      day: 'numeric',
    }) +
    ' at ' +
    date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    })
  );
}

export function ScheduleMessageModal({ isOpen, onClose, onSchedule }: ScheduleMessageModalProps) {
  const tz = getLocalTimeZone();
  const [selectedDate, setSelectedDate] = useState<DateValue | null>(null);
  const [selectedTime, setSelectedTime] = useState<TimeValue | null>(new Time(9, 0));

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

  const isCustomValid = customDate !== null && customDate > new Date();

  const handleSchedule = () => {
    if (customDate && isCustomValid) {
      onSchedule(customDate.toISOString());
      onClose();
      resetState();
    }
  };

  const resetState = () => {
    setSelectedDate(null);
    setSelectedTime(new Time(9, 0));
  };

  const handleClose = () => {
    onClose();
    resetState();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Custom time">
      <div className="space-y-4">
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
        </div>

        {customDate && isCustomValid && (
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Will be sent {formatScheduledDate(customDate)}
          </p>
        )}

        <div className="flex justify-end gap-2">
          <Button variant="secondary" onPress={handleClose}>
            Cancel
          </Button>
          <Button onPress={handleSchedule} isDisabled={!isCustomValid}>
            Schedule
          </Button>
        </div>
      </div>
    </Modal>
  );
}
