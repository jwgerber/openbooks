import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import { getApiURL } from "./util";

export interface IrcServer {
  elevatedUsers?: string[];
  regularUsers?: string[];
}

export interface Book {
  name: string;
  downloadLink: string;
  time: string;
}

export interface Settings {
  server: string;
  enableTls: boolean;
  channel: string;
  searchBot: string;
  userName: string;
  formatOrder: string[];
  downloadDir: string;
}

export const openbooksApi = createApi({
  baseQuery: fetchBaseQuery({
    baseUrl: getApiURL().href,
    credentials: "include",
    mode: "cors"
  }),
  tagTypes: ["books", "servers", "settings"],
  endpoints: (builder) => ({
    getServers: builder.query<string[], null>({
      query: () => `servers`,
      providesTags: ["servers"],
      transformResponse: (ircServers: IrcServer) => {
        // Any nick in the channel counts as online (matches server-side ranking).
        // Lowercased: IRC nicks are case-insensitive. Compare with isOnline().
        return [
          ...(ircServers.elevatedUsers ?? []),
          ...(ircServers.regularUsers ?? [])
        ].map((nick) => nick.toLowerCase());
      }
    }),
    getBooks: builder.query<Book[], null>({
      query: () => `library`,
      providesTags: ["books"]
    }),
    deleteBook: builder.mutation<null, string>({
      query: (book) => ({
        url: `library/${book}`,
        method: "DELETE"
      }),
      invalidatesTags: ["books"]
    }),
    getSettings: builder.query<Settings, null>({
      query: () => `settings`,
      providesTags: ["settings"]
    }),
    updateSettings: builder.mutation<Settings, Settings>({
      query: (settings) => ({
        url: `settings`,
        method: "PUT",
        body: settings
      }),
      invalidatesTags: ["settings", "servers"]
    })
  })
});

export const {
  useGetServersQuery,
  useGetBooksQuery,
  useDeleteBookMutation,
  useGetSettingsQuery,
  useUpdateSettingsMutation
} = openbooksApi;

// isOnline reports whether a server nick is in the (lowercased) online list.
export const isOnline = (servers: string[] | undefined, nick: string) =>
  servers?.includes(nick.toLowerCase()) ?? false;
