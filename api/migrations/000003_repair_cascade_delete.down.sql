ALTER TABLE repair_steps  DROP CONSTRAINT repair_steps_repair_id_fkey,
                          ADD  CONSTRAINT repair_steps_repair_id_fkey
                               FOREIGN KEY (repair_id) REFERENCES repairs(id);

ALTER TABLE parts         DROP CONSTRAINT parts_repair_id_fkey,
                          ADD  CONSTRAINT parts_repair_id_fkey
                               FOREIGN KEY (repair_id) REFERENCES repairs(id);

ALTER TABLE repair_notes  DROP CONSTRAINT repair_notes_repair_id_fkey,
                          ADD  CONSTRAINT repair_notes_repair_id_fkey
                               FOREIGN KEY (repair_id) REFERENCES repairs(id);

ALTER TABLE photos        DROP CONSTRAINT photos_repair_id_fkey,
                          ADD  CONSTRAINT photos_repair_id_fkey
                               FOREIGN KEY (repair_id) REFERENCES repairs(id);

ALTER TABLE resources     DROP CONSTRAINT resources_repair_id_fkey,
                          ADD  CONSTRAINT resources_repair_id_fkey
                               FOREIGN KEY (repair_id) REFERENCES repairs(id);