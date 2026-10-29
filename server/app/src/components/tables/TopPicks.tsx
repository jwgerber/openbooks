import {
  Badge,
  Group,
  Indicator,
  Paper,
  Stack,
  Text,
  Tooltip
} from "@mantine/core";
import { useGetServersQuery } from "../../state/api";
import { BookDetail } from "../../state/messages";
import { DownloadButton } from "./BookTable";

const TOP_PICK_COUNT = 5;

interface TopPicksProps {
  books: BookDetail[];
}

// Shows the highest-ranked results. Books arrive already sorted by the
// server, and copies of the same book are deliberately not collapsed.
export default function TopPicks({ books }: TopPicksProps) {
  const { data: servers } = useGetServersQuery(null);
  const picks = books.slice(0, TOP_PICK_COUNT);

  if (picks.length === 0) {
    return null;
  }

  return (
    <Stack spacing={6} mb="md" style={{ width: "100%" }}>
      <Text size="sm" weight={600}>
        Top picks
      </Text>
      {picks.map((book) => {
        const online = servers?.includes(book.server);
        return (
          <Paper key={book.full} withBorder p="xs" radius="md">
            <Group position="apart" noWrap>
              <Stack spacing={0} style={{ minWidth: 0 }}>
                <Text size="sm" weight={500} lineClamp={1}>
                  {book.title}
                </Text>
                <Text size="xs" color="dimmed" lineClamp={1}>
                  {book.author}
                </Text>
              </Stack>
              <Group spacing="sm" noWrap>
                <Badge size="sm" variant="light">
                  {book.format || "?"}
                </Badge>
                <Text size="xs" color="dimmed" style={{ width: 60 }}>
                  {book.size}
                </Text>
                <Tooltip label={online ? "Online" : "Offline"}>
                  <Indicator
                    zIndex={0}
                    position="middle-start"
                    offset={-10}
                    size={6}
                    color={online ? "green.6" : "gray"}>
                    <Text
                      size="xs"
                      style={{ width: 100, marginLeft: 8 }}
                      lineClamp={1}>
                      {book.server}
                    </Text>
                  </Indicator>
                </Tooltip>
                <DownloadButton book={book.full} />
              </Group>
            </Group>
          </Paper>
        );
      })}
    </Stack>
  );
}
