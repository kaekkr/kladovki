/** Storage statuses as worded in the user agreement (п. 3.1): «Свободно», «Забронировано», «Занято». */
export function storageStatusLabel(status: string): string {
  switch (status) {
    case 'free':
      return 'Свободно';
    case 'locked':
      return 'Забронировано';
    case 'occupied':
      return 'Занято';
    default:
      return status;
  }
}
