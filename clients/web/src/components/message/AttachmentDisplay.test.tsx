import { describe, it, expect, vi } from 'vitest';
import { render, screen, within, userEvent } from '../../test-utils';
import { AttachmentDisplay } from './AttachmentDisplay';
import type { Attachment } from '@feather/api-client';

// Mock the signedUrlCache so getCachedIfFresh returns URLs synchronously
vi.mock('../../lib/signedUrlCache', () => ({
  getCachedIfFresh: (fileId: string) => `/signed/${fileId}`,
  getUrl: (fileId: string) => Promise.resolve(`/signed/${fileId}`),
  getUrls: () => Promise.resolve(),
  invalidate: () => {},
}));

function makeImage(id: string, filename = `image-${id}.png`): Attachment {
  return {
    id,
    filename,
    content_type: 'image/png',
    size_bytes: 1024,
    url: `/uploads/${filename}`,
    created_at: new Date().toISOString(),
  };
}

function makeFile(id: string, filename = `doc-${id}.pdf`): Attachment {
  return {
    id,
    filename,
    content_type: 'application/pdf',
    size_bytes: 2048,
    url: `/uploads/${filename}`,
    created_at: new Date().toISOString(),
  };
}

describe('AttachmentDisplay', () => {
  it('renders nothing when attachments is empty', () => {
    const { container } = render(<AttachmentDisplay attachments={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders single image as thumbnail (not in grid)', () => {
    render(<AttachmentDisplay attachments={[makeImage('1')]} />);

    expect(screen.getByRole('button', { name: /View image-1\.png/ })).toBeInTheDocument();
    expect(screen.queryByTestId('image-grid')).not.toBeInTheDocument();
  });

  it('opens carousel when single image is clicked', async () => {
    const user = userEvent.setup();
    render(<AttachmentDisplay attachments={[makeImage('1', 'photo.png')]} />);

    await user.click(screen.getByRole('button', { name: /View photo\.png/ }));

    const dialog = screen.getByRole('dialog', { name: 'Image viewer' });
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByAltText('photo.png')).toBeInTheDocument();
    // Single image: no counter
    expect(screen.queryByTestId('carousel-counter')).not.toBeInTheDocument();
  });

  it('renders 2 images in a 2-column grid', () => {
    render(<AttachmentDisplay attachments={[makeImage('1'), makeImage('2')]} />);

    const grid = screen.getByTestId('image-grid');
    expect(grid).toBeInTheDocument();
    expect(grid.children).toHaveLength(2);
  });

  it('renders 4 images in a 2x2 grid', () => {
    const images = [makeImage('1'), makeImage('2'), makeImage('3'), makeImage('4')];
    render(<AttachmentDisplay attachments={images} />);

    const grid = screen.getByTestId('image-grid');
    expect(grid.children).toHaveLength(4);
    expect(screen.queryByTestId('overflow-count')).not.toBeInTheDocument();
  });

  it('renders 5+ images with +N overlay on 4th cell', () => {
    const images = Array.from({ length: 6 }, (_, i) => makeImage(String(i + 1)));
    render(<AttachmentDisplay attachments={images} />);

    const grid = screen.getByTestId('image-grid');
    expect(grid.children).toHaveLength(4);

    const overlay = screen.getByTestId('overflow-count');
    expect(overlay).toHaveTextContent('+3');
  });

  it('shows carousel counter when opened from grid', async () => {
    const user = userEvent.setup();
    const images = [
      makeImage('1', 'a.png'),
      makeImage('2', 'b.png'),
      makeImage('3', 'c.png'),
      makeImage('4', 'd.png'),
      makeImage('5', 'e.png'),
    ];
    render(<AttachmentDisplay attachments={images} />);

    await user.click(screen.getByRole('button', { name: /View a\.png/ }));

    const counter = screen.getByTestId('carousel-counter');
    expect(counter).toHaveTextContent('1 of 5');
  });

  it('navigates carousel with ArrowRight', async () => {
    const user = userEvent.setup();
    const images = [makeImage('1', 'a.png'), makeImage('2', 'b.png'), makeImage('3', 'c.png')];
    render(<AttachmentDisplay attachments={images} />);

    await user.click(screen.getByRole('button', { name: /View a\.png/ }));
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('1 of 3');

    await user.keyboard('{ArrowRight}');
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 of 3');
  });

  it('navigates carousel with ArrowLeft', async () => {
    const user = userEvent.setup();
    const images = [makeImage('1', 'a.png'), makeImage('2', 'b.png'), makeImage('3', 'c.png')];
    render(<AttachmentDisplay attachments={images} />);

    // Click second image
    await user.click(screen.getByRole('button', { name: /View b\.png/ }));
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 of 3');

    await user.keyboard('{ArrowLeft}');
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('1 of 3');
  });

  it('wraps from last to first with ArrowRight', async () => {
    const user = userEvent.setup();
    const images = [makeImage('1', 'a.png'), makeImage('2', 'b.png')];
    render(<AttachmentDisplay attachments={images} />);

    // Click second image (index 1)
    await user.click(screen.getByRole('button', { name: /View b\.png/ }));
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 of 2');

    await user.keyboard('{ArrowRight}');
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('1 of 2');
  });

  it('wraps from first to last with ArrowLeft', async () => {
    const user = userEvent.setup();
    const images = [makeImage('1', 'a.png'), makeImage('2', 'b.png')];
    render(<AttachmentDisplay attachments={images} />);

    // Click first image (index 0)
    await user.click(screen.getByRole('button', { name: /View a\.png/ }));
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('1 of 2');

    await user.keyboard('{ArrowLeft}');
    expect(screen.getByTestId('carousel-counter')).toHaveTextContent('2 of 2');
  });

  it('renders mixed attachments: images in grid, files separately', () => {
    const attachments = [
      makeImage('1'),
      makeImage('2'),
      makeImage('3'),
      makeFile('f1', 'report.pdf'),
      makeFile('f2', 'notes.txt'),
    ];
    render(<AttachmentDisplay attachments={attachments} />);

    // Images are in grid
    expect(screen.getByTestId('image-grid')).toBeInTheDocument();
    expect(screen.getByTestId('image-grid').children).toHaveLength(3);

    // Files rendered separately
    expect(screen.getByText('report.pdf')).toBeInTheDocument();
    expect(screen.getByText('notes.txt')).toBeInTheDocument();
  });

  it('renders only files when no images present', () => {
    render(<AttachmentDisplay attachments={[makeFile('1', 'doc.pdf')]} />);

    expect(screen.queryByTestId('image-grid')).not.toBeInTheDocument();
    expect(screen.getByText('doc.pdf')).toBeInTheDocument();
  });
});
