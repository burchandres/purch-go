--
-- PostgreSQL database dump
--

\restrict bfwM3YLA7htodXFKPFIrfrGkvcdpmglRaQcgLgJYjh8E4AB58XsLJVBxxujqpUX

-- Dumped from database version 18.0
-- Dumped by pg_dump version 18.0 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: items; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.items (id, user_id, access_token, name, transaction_cursor) VALUES ('C1-id', 'df901f07-7314-41b7-880c-1230038a328e', 'C1-token', 'Capital One', 'C1-cursor');
INSERT INTO public.items (id, user_id, access_token, name, transaction_cursor) VALUES ('WF-id', 'df901f07-7314-41b7-880c-1230038a328e', 'WF-token', 'Wells Fargo', 'WF-cursor');
INSERT INTO public.items (id, user_id, access_token, name, transaction_cursor) VALUES ('l33MWowj6xF91qWEeZrMho5XME16vBiZLae7e', 'b723a5fc-190f-41b0-95da-8cbc014cbafa', 'access-sandbox-197bccf6-8363-47a3-9b7d-ec915e3572e6', 'First Platypus Bank', 'CAESJThYWEJ3UG5BZ2V1bERWV0pxZ1A3dG9KVzQzUEVEcmhaWGVrRWsaDAiUrb7IBhCgicfGAiIMCJStvsgGEKCJx8YCKgwIlK2+yAYQoInHxgI=');


--
-- PostgreSQL database dump complete
--

\unrestrict bfwM3YLA7htodXFKPFIrfrGkvcdpmglRaQcgLgJYjh8E4AB58XsLJVBxxujqpUX

